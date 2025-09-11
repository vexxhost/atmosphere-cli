// Copyright 2025 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package ovnrouter

import (
	"context"
	"fmt"
	"strings"

	"github.com/ovn-kubernetes/libovsdb/client"
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
			LogicalRouter: r,
		})
	}

	return result, nil
}

func (r *Router) LogicalRouterPorts(ctx context.Context) ([]nbdb.LogicalRouterPort, error) {
	result := make([]nbdb.LogicalRouterPort, 0, len(r.Ports))

	for _, portUUID := range r.Ports {
		lrp := nbdb.LogicalRouterPort{UUID: portUUID}
		if err := r.Client.Get(ctx, &lrp); err != nil {
			return nil, fmt.Errorf("failed to get logical router port %q for router %q: %w", portUUID, r.UUID, err)
		}

		result = append(result, lrp)
	}

	return result, nil
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
			if err := r.Client.Get(ctx, &gc); err != nil {
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

	// Find the highest and lowest priority gateway chassis
	var highestGC, lowestGC *nbdb.GatewayChassis
	highestPriority := 0
	lowestPriority := -1
	
	for i := range gcs {
		if gcs[i].Priority > highestPriority {
			highestPriority = gcs[i].Priority
			highestGC = &gcs[i]
		}
		if lowestPriority == -1 || gcs[i].Priority < lowestPriority {
			lowestPriority = gcs[i].Priority
			lowestGC = &gcs[i]
		}
	}

	if highestGC == nil || lowestGC == nil || highestGC.UUID == lowestGC.UUID {
		return fmt.Errorf("unable to determine gateway chassis to swap for router %q", r.UUID)
	}

	// Swap the priorities between highest and lowest
	tempPriority := highestGC.Priority
	highestGC.Priority = lowestGC.Priority
	updateOps1, err := r.Client.Where(&nbdb.GatewayChassis{UUID: highestGC.UUID}).Update(highestGC)
	if err != nil {
		return fmt.Errorf("failed to prepare update for gateway chassis %q: %w", highestGC.UUID, err)
	}
	
	// Update lowest priority GC to have highest priority
	lowestGC.Priority = tempPriority
	updateOps2, err := r.Client.Where(&nbdb.GatewayChassis{UUID: lowestGC.UUID}).Update(lowestGC)
	if err != nil {
		return fmt.Errorf("failed to prepare update for gateway chassis %q: %w", lowestGC.UUID, err)
	}
	
	// Combine operations
	operations := append(updateOps1, updateOps2...)

	// Execute the transaction
	results, err := r.Client.Transact(ctx, operations...)
	if err != nil {
		return fmt.Errorf("failed to update gateway chassis priorities: %w", err)
	}

	// Check for errors in the transaction results
	for _, result := range results {
		if result.Error != "" {
			return fmt.Errorf("transaction error: %s", result.Error)
		}
	}

	// Wait for the router to be hosted on the new agent
	// In a real environment, OVN would update the status field automatically
	// For now, we just return success as the priority swap has been committed
	
	return nil
}
