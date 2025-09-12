// Copyright 2025 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package ovnrouter

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ovn-org/libovsdb/client"
	"github.com/ovn-org/libovsdb/model"
	"github.com/ovn-org/libovsdb/ovsdb"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/nbdb"
)

type Router struct {
	UUID string

	client.Client
	nbdb.LogicalRouter
}

func GetByName(ctx context.Context, client client.Client, name string) (*Router, error) {
	lrs := []nbdb.LogicalRouter{}
	if err := client.Where(&nbdb.LogicalRouter{Name: name}).List(ctx, &lrs); err != nil {
		return nil, fmt.Errorf("failed to get logical router %q: %w", name, err)
	}

	if len(lrs) == 0 {
		return nil, fmt.Errorf("logical router %q not found", name)
	}

	return &Router{
		UUID:          strings.TrimPrefix(lrs[0].Name, "neutron-"),
		Client:        client,
		LogicalRouter: lrs[0],
	}, nil
}

func List(ctx context.Context, client client.Client) ([]Router, error) {
	var routers []nbdb.LogicalRouter
	if err := client.List(ctx, &routers); err != nil {
		return nil, err
	}

	result := make([]Router, 0, len(routers))
	for _, r := range routers {
		result = append(result, Router{
			UUID:          strings.TrimPrefix(r.Name, "neutron-"),
			Client:        client,
			LogicalRouter: r,
		})
	}

	return result, nil
}

func (r *Router) LogicalRouterPorts(ctx context.Context) ([]nbdb.LogicalRouterPort, error) {
	result := make([]nbdb.LogicalRouterPort, 0, len(r.Ports))

	for _, portUUID := range r.Ports {
		lrp := nbdb.LogicalRouterPort{UUID: portUUID}
		if err := r.Get(ctx, &lrp); err != nil {
			return nil, fmt.Errorf("failed to get logical router port %q for router %q: %w", portUUID, r.UUID, err)
		}

		result = append(result, lrp)
	}

	return result, nil
}

func (r *Router) GetExternalIPs(ctx context.Context) ([]string, error) {
	lrps := []nbdb.LogicalRouterPort{}
	err := r.Client.WhereCache(func(lrp *nbdb.LogicalRouterPort) bool {
		if lrp.ExternalIDs == nil {
			return false
		}
		
		isExtGW, hasExtGW := lrp.ExternalIDs["neutron:is_ext_gw"]
		if !hasExtGW || isExtGW != "True" {
			return false
		}
		
		routerName, hasRouterName := lrp.ExternalIDs["neutron:router_name"]
		if !hasRouterName || routerName != r.UUID {
			return false
		}
		
		return true
	}).List(ctx, &lrps)

	if err != nil {
		return nil, fmt.Errorf("failed to get external gateway ports for router %q: %w", r.UUID, err)
	}
	
	var ips []string
	for _, lrp := range lrps {
		ips = append(ips, lrp.Networks...)
	}
	
	return ips, nil
}

func (r *Router) GatewayChassis(ctx context.Context) ([]nbdb.GatewayChassis, error) {
	lrps, err := r.LogicalRouterPorts(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]nbdb.GatewayChassis, 0)

	for _, lrp := range lrps {
		for _, gcUUID := range lrp.GatewayChassis {
			gc := nbdb.GatewayChassis{UUID: gcUUID}
			if err := r.Get(ctx, &gc); err != nil {
				return nil, fmt.Errorf("failed to get gateway chassis %q for logical router port %q: %w", gcUUID, lrp.UUID, err)
			}

			result = append(result, gc)
		}
	}

	return result, nil
}

func (r *Router) HostingAgent(ctx context.Context) (string, error) {
	lrps, err := r.LogicalRouterPorts(ctx)
	if err != nil {
		return "", err
	}

	if len(lrps) == 0 {
		return "", fmt.Errorf("no logical router ports found for router %q", r.UUID)
	}

	var agent string

	for _, lrp := range lrps {
		// NOTE(mnaser): Skip ports that are not external gateways.
		if len(lrp.GatewayChassis) == 0 {
			continue
		}

		agentChassis, ok := lrp.Status["hosting-chassis"]
		if !ok {
			return "", fmt.Errorf("no hosting-chassis found in status for logical router port %q", lrp.UUID)
		}

		if agent == "" {
			agent = agentChassis
		} else if agent != agentChassis {
			return "", fmt.Errorf("logical router ports for router %q are hosted on multiple agents: %q and %q", r.UUID, agent, agentChassis)
		}
	}

	if agent == "" {
		return "", fmt.Errorf("no hosting-chassis found for any logical router port of router %q", r.UUID)
	}

	return agent, nil
}

// Failover triggers a failover of the router from its current hosting gateway chassis
// to the next available one by swapping priorities between the highest and lowest.
//
// The failover mechanism swaps the highest priority (currently active) gateway chassis
// with the lowest priority gateway chassis. This simple approach works well for
// individual router failovers.
//
// After updating the priorities, the function waits for OVN to actually move the router
// to the new hosting chassis. The function polls every 500ms until the router is hosted
// on the expected chassis. The caller must provide a context with an appropriate deadline
// to prevent indefinite waiting.
//
// Note: When draining multiple nodes sequentially in a 3-node cluster, this approach
// may cause some routers to failover twice. For example:
//   - Initial: A=3 (active), B=2, C=1
//   - Drain A: C=3 (active), B=2, A=1 (swap A↔C)
//   - Drain B: No change (B not highest)
//   - Drain C: A=3 (active), B=2, C=1 (swap C↔A, router back on A)
//
// For optimal sequential node draining, a controller-aware orchestration layer
// should coordinate failovers to minimize total router movements.
//
// Example with 3 gateway chassis:
//
//	Initial: A(priority=3, active), B(priority=2), C(priority=1)
//	After failover: C(priority=3, active), B(priority=2), A(priority=1)
//
// The function requires at least 2 gateway chassis to perform a failover.
// Returns an error if no gateway chassis are found or if only one exists.
func (r *Router) Failover(ctx context.Context) error {
	gcs, err := r.GatewayChassis(ctx)
	if err != nil {
		return err
	}

	if len(gcs) == 0 {
		return fmt.Errorf("no gateway chassis found for router %q", r.UUID)
	}

	if len(gcs) == 1 {
		return fmt.Errorf("only one gateway chassis found for router %q, cannot failover", r.UUID)
	}

	// NOTE(mnaser): For simplicity, we sort the gateway chassis by priority from
	//               lowest to the highest.
	sort.Slice(gcs, func(i, j int) bool {
		return gcs[i].Priority < gcs[j].Priority
	})

	// NOTE(mnaser): The `nextGC` in this case is the one with the lowest priority
	//               which will become the active one after the failover.  The `currentGC`
	//               is the one with the highest priority which is currently active.
	nextGC := &gcs[0]
	currentGC := &gcs[len(gcs)-1]

	if nextGC.UUID == currentGC.UUID {
		return fmt.Errorf("unable to determine gateway chassis to swap for router %q", r.UUID)
	}

	// NOTE(mnaser): Swap priorities between the current active and the next one.  Once
	//               we do this, OVN should automatically move the router to the new
	//               hosting chassis.
	updates := []model.Model{
		&nbdb.GatewayChassis{
			UUID:     currentGC.UUID,
			Priority: nextGC.Priority,
		},
		&nbdb.GatewayChassis{
			UUID:     nextGC.UUID,
			Priority: currentGC.Priority,
		},
	}

	var operations []ovsdb.Operation
	for _, update := range updates {
		ops, err := r.Client.Where(update).Update(update)
		if err != nil {
			return fmt.Errorf("failed to prepare update for gateway chassis: %w", err)
		}

		operations = append(operations, ops...)
	}

	results, err := r.Transact(ctx, operations...)
	if err != nil {
		return fmt.Errorf("failed to update gateway chassis priorities: %w", err)
	}

	if _, err := ovsdb.CheckOperationResults(results, operations); err != nil {
		return err
	}

	// NOTE(mnaser): The hosting agent should be the one in the `nextGC` now.
	expectedHost := nextGC.ChassisName

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("failed waiting for router %q to failover to %q: %w", r.UUID, expectedHost, ctx.Err())
		case <-ticker.C:
			currentHost, err := r.HostingAgent(ctx)
			if err != nil {
				continue
			}

			if currentHost == expectedHost {
				return nil
			}
		}
	}
}
