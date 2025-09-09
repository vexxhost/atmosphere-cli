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
		nbData   []libovsdb.TestData
		expected []string
	}{
		{
			name: "single router",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name: "neutron-" + testRouterUUID,
				},
			},
			expected: []string{testRouterUUID},
		},
		{
			name: "multiple routers",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name: "neutron-" + testRouterUUID,
				},
				&nbdb.LogicalRouter{
					Name: "neutron-" + testRouterUUID2,
				},
			},
			expected: []string{testRouterUUID, testRouterUUID2},
		},
		{
			name:     "no routers",
			nbData:   []libovsdb.TestData{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			nbClient, cleanup := setupTestHarnessForTest(t, tt.nbData)
			t.Cleanup(cleanup.Cleanup)

			routers, err := List(ctx, nbClient)
			require.NoError(t, err)

			require.Len(t, routers, len(tt.expected))
			for i, expectedUUID := range tt.expected {
				assert.Equal(t, expectedUUID, routers[i].UUID)
			}
		})
	}
}

func TestRouter_LogicalRouterPorts(t *testing.T) {
	tests := []struct {
		name        string
		nbData      []libovsdb.TestData
		expected    []*nbdb.LogicalRouterPort
		expectError bool
		errorMsg    string
	}{
		{
			name: "basic",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{testPortUUID1, testPortUUID2},
				},
				&nbdb.LogicalRouterPort{UUID: testPortUUID1, Name: "lrp-1"},
				&nbdb.LogicalRouterPort{UUID: testPortUUID2, Name: "lrp-2"},
			},
			expected: []*nbdb.LogicalRouterPort{
				{UUID: testPortUUID1, Name: "lrp-1"},
				{UUID: testPortUUID2, Name: "lrp-2"},
			},
		},
		{
			name: "no ports",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{},
				},
			},
			expected: []*nbdb.LogicalRouterPort{},
		},
		{
			name: "multiple routers",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{testPortUUID1},
				},
				&nbdb.LogicalRouter{
					Name:  "neutron-" + testRouterUUID2,
					Ports: []string{testPortUUID2},
				},
				&nbdb.LogicalRouterPort{UUID: testPortUUID1, Name: "lrp-1"},
				&nbdb.LogicalRouterPort{UUID: testPortUUID2, Name: "lrp-2"},
			},
			expected: []*nbdb.LogicalRouterPort{
				{UUID: testPortUUID1, Name: "lrp-1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			nbClient, cleanup := setupTestHarnessForTest(t, tt.nbData)
			t.Cleanup(cleanup.Cleanup)

			router, err := GetByName(ctx, nbClient, "neutron-"+testRouterUUID)
			require.NoError(t, err)

			ports, err := router.LogicalRouterPorts(ctx, nbClient)
			require.NoError(t, err)

			require.Len(t, ports, len(tt.expected))
			for i, expected := range tt.expected {
				assert.Equal(t, expected.UUID, ports[i].UUID)
				assert.Equal(t, expected.Name, ports[i].Name)
			}
		})
	}
}
func TestRouter_GatewayChassis(t *testing.T) {
	tests := []struct {
		name     string
		nbData   []libovsdb.TestData
		expected []nbdb.GatewayChassis
	}{
		{
			name: "single port with single gateway chassis",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{testPortUUID1},
				},
				&nbdb.LogicalRouterPort{
					UUID:           testPortUUID1,
					Name:           "lrp-1",
					GatewayChassis: []string{testChassisUUID},
				},
				&nbdb.GatewayChassis{
					UUID:     testChassisUUID,
					Name:     "gwc-1",
					Priority: 1,
				},
			},
			expected: []nbdb.GatewayChassis{
				{UUID: testChassisUUID, Name: "gwc-1", Priority: 1},
			},
		},
		{
			name: "single port with multiple gateway chassis",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{testPortUUID1},
				},
				&nbdb.LogicalRouterPort{
					UUID:           testPortUUID1,
					Name:           "lrp-1",
					GatewayChassis: []string{testChassisUUID, testChassisUUID2},
				},
				&nbdb.GatewayChassis{
					UUID:     testChassisUUID,
					Name:     "gwc-1",
					Priority: 1,
				},
				&nbdb.GatewayChassis{
					UUID:     testChassisUUID2,
					Name:     "gwc-2",
					Priority: 2,
				},
			},
			expected: []nbdb.GatewayChassis{
				{UUID: testChassisUUID, Name: "gwc-1", Priority: 1},
				{UUID: testChassisUUID2, Name: "gwc-2", Priority: 2},
			},
		},
		{
			name: "multiple ports with one holding gateway chassis",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{testPortUUID1, testPortUUID2},
				},
				&nbdb.LogicalRouterPort{
					UUID:           testPortUUID1,
					Name:           "lrp-1",
					GatewayChassis: []string{testChassisUUID, testChassisUUID2},
				},
				&nbdb.LogicalRouterPort{
					UUID:           testPortUUID2,
					Name:           "lrp-2",
					GatewayChassis: []string{},
				},
				&nbdb.GatewayChassis{
					UUID:     testChassisUUID,
					Name:     "gwc-1",
					Priority: 1,
				},
				&nbdb.GatewayChassis{
					UUID:     testChassisUUID2,
					Name:     "gwc-2",
					Priority: 2,
				},
			},
			expected: []nbdb.GatewayChassis{
				{UUID: testChassisUUID, Name: "gwc-1", Priority: 1},
				{UUID: testChassisUUID2, Name: "gwc-2", Priority: 2},
			},
		},
		{
			name: "no gateway chassis on port",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{testPortUUID1},
				},
				&nbdb.LogicalRouterPort{
					UUID:           testPortUUID1,
					Name:           "lrp-1",
					GatewayChassis: []string{},
				},
			},
			expected: []nbdb.GatewayChassis{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			nbClient, cleanup := setupTestHarnessForTest(t, tt.nbData)
			t.Cleanup(cleanup.Cleanup)

			router, err := GetByName(ctx, nbClient, "neutron-"+testRouterUUID)
			require.NoError(t, err)

			gcs, err := router.GatewayChassis(ctx, nbClient)
			require.NoError(t, err)

			require.Len(t, gcs, len(tt.expected))
			for i, expected := range tt.expected {
				assert.Equal(t, expected.UUID, gcs[i].UUID)
				assert.Equal(t, expected.Name, gcs[i].Name)
			}
		})
	}
}

func TestRouter_HostingAgent(t *testing.T) {
	tests := []struct {
		name          string
		nbData        []libovsdb.TestData
		router        *nbdb.LogicalRouter
		lrps          []*nbdb.LogicalRouterPort
		expectedAgent string
		expectError   bool
		errorContains string
	}{
		{
			name: "single agent hosting all ports",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{testPortUUID1, testPortUUID2},
				},
				&nbdb.LogicalRouterPort{
					UUID:   testPortUUID1,
					Name:   "lrp-1",
					Status: map[string]string{"hosting-chassis": testChassisUUID},
				},
				&nbdb.LogicalRouterPort{
					UUID:   testPortUUID2,
					Name:   "lrp-2",
					Status: map[string]string{"hosting-chassis": testChassisUUID},
				},
			},
			expectedAgent: testChassisUUID,
		},
		{
			name: "multiple agents hosting different ports",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{testPortUUID1, testPortUUID2},
				},
				&nbdb.LogicalRouterPort{
					UUID:   testPortUUID1,
					Name:   "lrp-1",
					Status: map[string]string{"hosting-chassis": testChassisUUID},
				},
				&nbdb.LogicalRouterPort{
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
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{},
				},
			},
			expectError:   true,
			errorContains: "no logical router ports found",
		},
		{
			name: "port missing hosting-chassis status",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{testPortUUID1},
				},
				&nbdb.LogicalRouterPort{
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

			nbClient, cleanup := setupTestHarnessForTest(t, tt.nbData)
			t.Cleanup(cleanup.Cleanup)

			router, err := GetByName(ctx, nbClient, "neutron-"+testRouterUUID)
			require.NoError(t, err)

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
