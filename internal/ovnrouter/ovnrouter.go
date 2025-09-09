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

	*nbdb.LogicalRouter
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
			LogicalRouter: &r,
		})
	}

	return result, nil
}

func (r *Router) HostingAgent(ctx context.Context, client client.Client) (string, error) {
	if len(r.Ports) == 0 {
		return "", fmt.Errorf("no logical router ports found for router %q", r.UUID)
	}

	var agent string

	for _, portUUID := range r.Ports {
		lrp := &nbdb.LogicalRouterPort{UUID: portUUID}
		if err := client.Get(ctx, lrp); err != nil {
			return "", fmt.Errorf("failed to get logical router port %q for router %q: %w", portUUID, r.UUID, err)
		}

		agentChassis, ok := lrp.Status["hosting-chassis"]
		if !ok {
			return "", fmt.Errorf("no hosting-chassis found in status for logical router port %q", portUUID)
		}

		if agent == "" {
			agent = agentChassis
		} else if agent != agentChassis {
			return "", fmt.Errorf("logical router ports for router %q are hosted on multiple agents: %q and %q", r.UUID, agent, agentChassis)
		}
	}

	return agent, nil
}
