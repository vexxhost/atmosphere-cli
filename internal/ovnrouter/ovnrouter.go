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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	apiv1alpha1 "github.com/vexxhost/atmosphere/apis/v1alpha1"
)

// GetByName retrieves a router by its name from OVN
func GetByName(ctx context.Context, c client.Client, name string) (*apiv1alpha1.Router, *nbdb.LogicalRouter, error) {
	lrs := []nbdb.LogicalRouter{}
	if err := c.Where(&nbdb.LogicalRouter{Name: name}).List(ctx, &lrs); err != nil {
		return nil, nil, fmt.Errorf("failed to get logical router %q: %w", name, err)
	}

	if len(lrs) == 0 {
		return nil, nil, fmt.Errorf("logical router %q not found", name)
	}

	uuid := strings.TrimPrefix(lrs[0].Name, "neutron-")
	// Try to get the router name from ExternalIDs, fallback to UUID
	routerName := uuid
	if lrs[0].ExternalIDs != nil {
		if name, ok := lrs[0].ExternalIDs["neutron:router_name"]; ok && name != "" {
			routerName = name
		}
	}

	router := &apiv1alpha1.Router{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Router",
			APIVersion: "atmosphere.vexxhost.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: routerName,
			UID:  types.UID(uuid),
		},
	}

	// Fetch external IPs
	if externalIPs, err := GetExternalIPs(ctx, c, &lrs[0]); err == nil {
		router.Status.ExternalIPs = externalIPs
	}

	// Fetch hosting agent
	if hostingAgent, err := GetHostingAgent(ctx, c, &lrs[0]); err == nil {
		router.Status.Agent = hostingAgent
	}

	return router, &lrs[0], nil
}

// List retrieves all routers from OVN
func List(ctx context.Context, c client.Client) (*apiv1alpha1.RouterList, error) {
	var routers []nbdb.LogicalRouter
	if err := c.List(ctx, &routers); err != nil {
		return nil, err
	}

	result := &apiv1alpha1.RouterList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RouterList",
			APIVersion: "atmosphere.vexxhost.com/v1alpha1",
		},
		Items: make([]apiv1alpha1.Router, 0, len(routers)),
	}

	for _, lr := range routers {
		uuid := strings.TrimPrefix(lr.Name, "neutron-")
		// Try to get the router name from ExternalIDs, fallback to UUID
		routerName := uuid
		if lr.ExternalIDs != nil {
			if name, ok := lr.ExternalIDs["neutron:router_name"]; ok && name != "" {
				routerName = name
			}
		}

		router := apiv1alpha1.Router{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Router",
				APIVersion: "atmosphere.vexxhost.com/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: routerName,
				UID:  types.UID(uuid),
			},
		}

		// Fetch external IPs for this router
		if externalIPs, err := GetExternalIPs(ctx, c, &lr); err == nil {
			router.Status.ExternalIPs = externalIPs
		}

		// Fetch hosting agent for this router
		if hostingAgent, err := GetHostingAgent(ctx, c, &lr); err == nil {
			router.Status.Agent = hostingAgent
		}

		result.Items = append(result.Items, router)
	}

	return result, nil
}

// GetLogicalRouterPorts retrieves all logical router ports for a given router
func GetLogicalRouterPorts(ctx context.Context, c client.Client, lr *nbdb.LogicalRouter) ([]nbdb.LogicalRouterPort, error) {
	result := make([]nbdb.LogicalRouterPort, 0, len(lr.Ports))

	for _, portUUID := range lr.Ports {
		lrp := nbdb.LogicalRouterPort{UUID: portUUID}
		if err := c.Get(ctx, &lrp); err != nil {
			return nil, fmt.Errorf("failed to get logical router port %q for router %q: %w", portUUID, lr.Name, err)
		}

		result = append(result, lrp)
	}

	return result, nil
}

// GetExternalIPs retrieves the external IP addresses for a router
func GetExternalIPs(ctx context.Context, c client.Client, lr *nbdb.LogicalRouter) ([]string, error) {
	routerUUID := strings.TrimPrefix(lr.Name, "neutron-")

	lrps := []nbdb.LogicalRouterPort{}
	err := c.WhereCache(func(lrp *nbdb.LogicalRouterPort) bool {
		if lrp.ExternalIDs == nil {
			return false
		}

		isExtGW, hasExtGW := lrp.ExternalIDs["neutron:is_ext_gw"]
		if !hasExtGW || isExtGW != "True" {
			return false
		}

		routerName, hasRouterName := lrp.ExternalIDs["neutron:router_name"]
		if !hasRouterName || routerName != routerUUID {
			return false
		}

		return true
	}).List(ctx, &lrps)

	if err != nil {
		return nil, fmt.Errorf("failed to get external gateway ports for router %q: %w", routerUUID, err)
	}

	var ips []string
	for _, lrp := range lrps {
		ips = append(ips, lrp.Networks...)
	}

	return ips, nil
}

// GetGatewayChassis retrieves all gateway chassis for a router
func GetGatewayChassis(ctx context.Context, c client.Client, lr *nbdb.LogicalRouter) ([]nbdb.GatewayChassis, error) {
	lrps, err := GetLogicalRouterPorts(ctx, c, lr)
	if err != nil {
		return nil, err
	}

	result := make([]nbdb.GatewayChassis, 0)

	for _, lrp := range lrps {
		for _, gcUUID := range lrp.GatewayChassis {
			gc := nbdb.GatewayChassis{UUID: gcUUID}
			if err := c.Get(ctx, &gc); err != nil {
				return nil, fmt.Errorf("failed to get gateway chassis %q for logical router port %q: %w", gcUUID, lrp.UUID, err)
			}

			result = append(result, gc)
		}
	}

	return result, nil
}

// GetHostingAgent retrieves the name of the agent hosting the router
func GetHostingAgent(ctx context.Context, c client.Client, lr *nbdb.LogicalRouter) (string, error) {
	lrps, err := GetLogicalRouterPorts(ctx, c, lr)
	if err != nil {
		return "", err
	}

	if len(lrps) == 0 {
		return "", fmt.Errorf("no logical router ports found for router %q", lr.Name)
	}

	var agent string

	for _, lrp := range lrps {
		// Skip ports that are not external gateways
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
			return "", fmt.Errorf("logical router ports for router %q are hosted on multiple agents: %q and %q", lr.Name, agent, agentChassis)
		}
	}

	if agent == "" {
		return "", fmt.Errorf("no hosting-chassis found for any logical router port of router %q", lr.Name)
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
func Failover(ctx context.Context, c client.Client, router *apiv1alpha1.Router) error {
	// Get the OVN logical router using the UID
	lrs := []nbdb.LogicalRouter{}
	if err := c.Where(&nbdb.LogicalRouter{Name: "neutron-" + string(router.UID)}).List(ctx, &lrs); err != nil {
		return fmt.Errorf("failed to get logical router %q: %w", router.UID, err)
	}

	if len(lrs) == 0 {
		return fmt.Errorf("logical router %q not found", router.UID)
	}

	lr := &lrs[0]

	gcs, err := GetGatewayChassis(ctx, c, lr)
	if err != nil {
		return err
	}

	if len(gcs) == 0 {
		return fmt.Errorf("no gateway chassis found for router %q", router.UID)
	}

	if len(gcs) == 1 {
		return fmt.Errorf("only one gateway chassis found for router %q, cannot failover", router.UID)
	}

	// Sort the gateway chassis by priority from lowest to the highest
	sort.Slice(gcs, func(i, j int) bool {
		return gcs[i].Priority < gcs[j].Priority
	})

	// The `nextGC` is the one with the lowest priority which will become active
	// The `currentGC` is the one with the highest priority which is currently active
	nextGC := &gcs[0]
	currentGC := &gcs[len(gcs)-1]

	if nextGC.UUID == currentGC.UUID {
		return fmt.Errorf("unable to determine gateway chassis to swap for router %q", router.UID)
	}

	// Swap priorities between the current active and the next one
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
		ops, err := c.Where(update).Update(update)
		if err != nil {
			return fmt.Errorf("failed to prepare update for gateway chassis: %w", err)
		}

		operations = append(operations, ops...)
	}

	results, err := c.Transact(ctx, operations...)
	if err != nil {
		return fmt.Errorf("failed to update gateway chassis priorities: %w", err)
	}

	if _, err := ovsdb.CheckOperationResults(results, operations); err != nil {
		return err
	}

	// The hosting agent should be the one in the `nextGC` now
	expectedHost := nextGC.ChassisName

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("failed waiting for router %q to failover to %q: %w", router.UID, expectedHost, ctx.Err())
		case <-ticker.C:
			currentHost, err := GetHostingAgent(ctx, c, lr)
			if err != nil {
				continue
			}

			if currentHost == expectedHost {
				return nil
			}
		}
	}
}
