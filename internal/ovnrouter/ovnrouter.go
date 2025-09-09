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

func (r *Router) LogicalRouterPorts(ctx context.Context, client client.Client) ([]nbdb.LogicalRouterPort, error) {
	result := make([]nbdb.LogicalRouterPort, 0, len(r.Ports))

	for _, portUUID := range r.Ports {
		lrp := nbdb.LogicalRouterPort{UUID: portUUID}
		if err := client.Get(ctx, &lrp); err != nil {
			return nil, fmt.Errorf("failed to get logical router port %q for router %q: %w", portUUID, r.UUID, err)
		}

		result = append(result, lrp)
	}

	return result, nil
}

func (r *Router) GatewayChassis(ctx context.Context, client client.Client) ([]nbdb.GatewayChassis, error) {
	lrps, err := r.LogicalRouterPorts(ctx, client)
	if err != nil {
		return nil, err
	}

	result := make([]nbdb.GatewayChassis, 0)

	for _, lrp := range lrps {
		for _, gcUUID := range lrp.GatewayChassis {
			gc := nbdb.GatewayChassis{UUID: gcUUID}
			if err := client.Get(ctx, &gc); err != nil {
				return nil, fmt.Errorf("failed to get gateway chassis %q for logical router port %q: %w", gcUUID, lrp.UUID, err)
			}

			result = append(result, gc)
		}
	}

	return result, nil
}

func (r *Router) HostingAgent(ctx context.Context, client client.Client) (string, error) {
	lrps, err := r.LogicalRouterPorts(ctx, client)
	if err != nil {
		return "", err
	}

	if len(lrps) == 0 {
		return "", fmt.Errorf("no logical router ports found for router %q", r.UUID)
	}

	var agent string

	for _, lrp := range lrps {
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

	return agent, nil
}
