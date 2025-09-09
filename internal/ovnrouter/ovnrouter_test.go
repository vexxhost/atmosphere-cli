// Copyright 2025 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package ovnrouter

import (
	"context"
	"testing"

	"github.com/ovn-kubernetes/libovsdb/client"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/nbdb"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/testing/libovsdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRouterUUID   = "266b4831-c71b-46f0-bfdc-a0bd189db632"
	testRouterUUID2  = "366b4831-c71b-46f0-bfdc-a0bd189db632"
	testPortUUID1    = "1637f6e5-360b-47d0-b867-04c6f155c697"
	testPortUUID2    = "1a3a3c85-bf28-4285-9a64-44685309b49d"
	testChassisUUID  = "aa3fd293-3f8c-42f9-9d72-4afa984727b3"
	testChassisUUID2 = "bb4fd293-3f8c-42f9-9d72-4afa984727b3"
)

func buildTestData(routers []*nbdb.LogicalRouter, lrps []*nbdb.LogicalRouterPort) []libovsdb.TestData {
	nbData := make([]libovsdb.TestData, 0, len(routers)+len(lrps))

	for _, router := range routers {
		nbData = append(nbData, router)
	}

	for _, lrp := range lrps {
		nbData = append(nbData, lrp)
	}

	return nbData
}

func setupTestHarnessForTest(t *testing.T, nbData []libovsdb.TestData) (client.Client, *libovsdb.Context) {
	t.Helper()

	nbClient, cleanup, err := libovsdb.NewNBTestHarness(libovsdb.TestSetup{
		NBData: nbData,
	}, nil)
	require.NoError(t, err)

	return nbClient, cleanup
}

func TestList(t *testing.T) {
	tests := []struct {
		name     string
		routers  []*nbdb.LogicalRouter
		lrps     []*nbdb.LogicalRouterPort
		expected []Router
	}{
		{
			name: "single router",
			routers: []*nbdb.LogicalRouter{
				{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{testPortUUID1, testPortUUID2},
				},
			},
			lrps: []*nbdb.LogicalRouterPort{
				{
					UUID:   testPortUUID1,
					Name:   "lrp-1",
					Status: map[string]string{"hosting-chassis": testChassisUUID},
				},
				{
					UUID:   testPortUUID2,
					Name:   "lrp-2",
					Status: map[string]string{"hosting-chassis": testChassisUUID},
				},
			},
			expected: []Router{
				{
					UUID: testRouterUUID,
					LogicalRouter: &nbdb.LogicalRouter{
						Name:  "neutron-" + testRouterUUID,
						Ports: []string{testPortUUID1, testPortUUID2},
					},
				},
			},
		},
		{
			name: "multiple routers",
			routers: []*nbdb.LogicalRouter{
				{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{testPortUUID1},
				},
				{
					Name:  "neutron-" + testRouterUUID2,
					Ports: []string{testPortUUID2},
				},
			},
			lrps: []*nbdb.LogicalRouterPort{
				{
					UUID:   testPortUUID1,
					Name:   "lrp-1",
					Status: map[string]string{"hosting-chassis": testChassisUUID},
				},
				{
					UUID:   testPortUUID2,
					Name:   "lrp-2",
					Status: map[string]string{"hosting-chassis": testChassisUUID2},
				},
			},
			expected: []Router{
				{
					UUID: testRouterUUID,
					LogicalRouter: &nbdb.LogicalRouter{
						Name:  "neutron-" + testRouterUUID,
						Ports: []string{testPortUUID1},
					},
				},
				{
					UUID: testRouterUUID2,
					LogicalRouter: &nbdb.LogicalRouter{
						Name:  "neutron-" + testRouterUUID2,
						Ports: []string{testPortUUID2},
					},
				},
			},
		},
		{
			name:     "no routers",
			routers:  []*nbdb.LogicalRouter{},
			lrps:     []*nbdb.LogicalRouterPort{},
			expected: []Router{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			nbClient, cleanup := setupTestHarnessForTest(t, buildTestData(tt.routers, tt.lrps))
			t.Cleanup(cleanup.Cleanup)

			routers, err := List(ctx, nbClient)
			require.NoError(t, err)

			require.Len(t, routers, len(tt.expected))
			for i, expectedRouter := range tt.expected {
				assert.Equal(t, expectedRouter.UUID, routers[i].UUID)
				assert.Equal(t, expectedRouter.LogicalRouter.Name, routers[i].LogicalRouter.Name)
				assert.Equal(t, expectedRouter.LogicalRouter.Ports, routers[i].LogicalRouter.Ports)
			}
		})
	}
}

func TestRouter_HostingAgent(t *testing.T) {
	tests := []struct {
		name          string
		router        *nbdb.LogicalRouter
		lrps          []*nbdb.LogicalRouterPort
		expectedAgent string
		expectError   bool
		errorContains string
	}{
		{
			name: "single agent hosting all ports",
			router: &nbdb.LogicalRouter{
				Name:  "neutron-" + testRouterUUID,
				Ports: []string{testPortUUID1, testPortUUID2},
			},
			lrps: []*nbdb.LogicalRouterPort{
				{
					UUID:   testPortUUID1,
					Name:   "lrp-1",
					Status: map[string]string{"hosting-chassis": testChassisUUID},
				},
				{
					UUID:   testPortUUID2,
					Name:   "lrp-2",
					Status: map[string]string{"hosting-chassis": testChassisUUID},
				},
			},
			expectedAgent: testChassisUUID,
		},
		{
			name: "multiple agents hosting different ports",
			router: &nbdb.LogicalRouter{
				Name:  "neutron-" + testRouterUUID,
				Ports: []string{testPortUUID1, testPortUUID2},
			},
			lrps: []*nbdb.LogicalRouterPort{
				{
					UUID:   testPortUUID1,
					Name:   "lrp-1",
					Status: map[string]string{"hosting-chassis": testChassisUUID},
				},
				{
					UUID:   testPortUUID2,
					Name:   "lrp-2",
					Status: map[string]string{"hosting-chassis": testChassisUUID2},
				},
			},
			expectError:   true,
			errorContains: "hosted on multiple agents",
		},
		{
			name: "no ports",
			router: &nbdb.LogicalRouter{
				Name:  "neutron-" + testRouterUUID,
				Ports: []string{},
			},
			lrps:          []*nbdb.LogicalRouterPort{},
			expectError:   true,
			errorContains: "no logical router ports found",
		},
		{
			name: "port missing hosting-chassis status",
			router: &nbdb.LogicalRouter{
				Name:  "neutron-" + testRouterUUID,
				Ports: []string{testPortUUID1},
			},
			lrps: []*nbdb.LogicalRouterPort{
				{
					UUID:   testPortUUID1,
					Name:   "lrp-1",
					Status: map[string]string{},
				},
			},
			expectError:   true,
			errorContains: "no hosting-chassis found in status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			nbClient, cleanup := setupTestHarnessForTest(t, buildTestData([]*nbdb.LogicalRouter{tt.router}, tt.lrps))
			t.Cleanup(cleanup.Cleanup)

			router := &Router{
				UUID: testRouterUUID,
				LogicalRouter: &nbdb.LogicalRouter{
					Ports: tt.router.Ports,
				},
			}

			agent, err := router.HostingAgent(ctx, nbClient)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Empty(t, agent)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedAgent, agent)
			}
		})
	}
}
