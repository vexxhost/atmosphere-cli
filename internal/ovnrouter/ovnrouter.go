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
	"k8s.io/utils/ptr"

	apiv1alpha1 "github.com/vexxhost/atmosphere/apis/v1alpha1"
)

func convertToRouter(ctx context.Context, c client.Client, lr *nbdb.LogicalRouter) (*apiv1alpha1.Router, error) {
	routerUUID := strings.TrimPrefix(lr.Name, "neutron-")
	uuid := types.UID(routerUUID)

	routerName := string(uuid)
	if lr.ExternalIDs != nil {
		if name, ok := lr.ExternalIDs["neutron:router_name"]; ok && name != "" {
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
		Status: apiv1alpha1.RouterStatus{
			InternalUUID: ptr.To(types.UID(lr.UUID)),
		},
	}

	for _, portUUID := range lr.Ports {
		lrp := nbdb.LogicalRouterPort{UUID: portUUID}
		if err := c.Get(ctx, &lrp); err != nil {
			return nil, fmt.Errorf("failed to get logical router port %q for router %q: %w", portUUID, lr.Name, err)
		}

		if lrp.ExternalIDs["neutron:is_ext_gw"] == "True" {
			router.Status.ExternalIPs = append(router.Status.ExternalIPs, lrp.Networks...)
			router.Status.Agent = lrp.Status["hosting-chassis"]
		}

		router.Status.Ports = append(router.Status.Ports, apiv1alpha1.RouterPortInfo{
			UUID:         types.UID(strings.TrimPrefix(lrp.Name, "lrp-")),
			InternalUUID: ptr.To(types.UID(lrp.UUID)),
			IsGateway:    lrp.ExternalIDs["neutron:is_ext_gw"] == "True",
		})
	}

	return router, nil
}

func GetByUUID(ctx context.Context, c client.Client, uuid types.UID) (*apiv1alpha1.Router, error) {
	lrs := []nbdb.LogicalRouter{}
	if err := c.Where(&nbdb.LogicalRouter{Name: fmt.Sprintf("neutron-%s", uuid)}).List(ctx, &lrs); err != nil {
		return nil, fmt.Errorf("failed to get router %q: %w", uuid, err)
	}

	if len(lrs) == 0 {
		return nil, fmt.Errorf("router %q not found", uuid)
	}

	return convertToRouter(ctx, c, &lrs[0])
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
		router, err := convertToRouter(ctx, c, &lr)
		if err != nil {
			continue
		}

		result.Items = append(result.Items, *router)
	}

	return result, nil
}

// GetLogicalRouterPorts retrieves all logical router ports for a given router
func GetLogicalRouterPorts(ctx context.Context, c client.Client, router *apiv1alpha1.Router) ([]nbdb.LogicalRouterPort, error) {
	result := make([]nbdb.LogicalRouterPort, 0, len(router.Status.Ports))

	for _, port := range router.Status.Ports {
		lrp := nbdb.LogicalRouterPort{UUID: string(*port.InternalUUID)}
		if err := c.Get(ctx, &lrp); err != nil {
			return nil, fmt.Errorf("failed to get logical router port %q for router %q: %w", port.UUID, router.UID, err)
		}

		result = append(result, lrp)
	}

	return result, nil
}

// GetGatewayChassis retrieves all gateway chassis for a router
func GetGatewayChassis(ctx context.Context, c client.Client, router *apiv1alpha1.Router) ([]nbdb.GatewayChassis, error) {
	lrps, err := GetLogicalRouterPorts(ctx, c, router)
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
func GetHostingAgent(ctx context.Context, c client.Client, router *apiv1alpha1.Router) (string, error) {
	lrps, err := GetLogicalRouterPorts(ctx, c, router)
	if err != nil {
		return "", err
	}

	if len(lrps) == 0 {
		return "", fmt.Errorf("no logical router ports found for router %q", router.Name)
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
			return "", fmt.Errorf("logical router ports for router %q are hosted on multiple agents: %q and %q", router.UID, agent, agentChassis)
		}
	}

	if agent == "" {
		return "", fmt.Errorf("no hosting-chassis found for any logical router port of router %q", router.UID)
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
	gcs, err := GetGatewayChassis(ctx, c, router)
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
			currentHost, err := GetHostingAgent(ctx, c, router)
			if err != nil {
				continue
			}

			if currentHost == expectedHost {
				return nil
			}
		}
	}
}
