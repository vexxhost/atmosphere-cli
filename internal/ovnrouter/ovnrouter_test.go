// Copyright 2025 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package ovnrouter

import (
	"context"
	"testing"
	"time"

	"github.com/ovn-org/libovsdb/cache"
	"github.com/ovn-org/libovsdb/client"
	"github.com/ovn-org/libovsdb/model"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/nbdb"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/testing/libovsdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
)

const (
	testRouterUUID   = "266b4831-c71b-46f0-bfdc-a0bd189db632"
	testRouterUUID2  = "366b4831-c71b-46f0-bfdc-a0bd189db632"
	testPortUUID1    = "1637f6e5-360b-47d0-b867-04c6f155c697"
	testPortUUID2    = "1a3a3c85-bf28-4285-9a64-44685309b49d"
	testChassisUUID  = "aa3fd293-3f8c-42f9-9d72-4afa984727b3"
	testChassisUUID2 = "bb4fd293-3f8c-42f9-9d72-4afa984727b3"
	testChassisUUID3 = "cc4fd293-3f8c-42f9-9d72-4afa984727b3"
)

func setupTestHarnessForTest(t *testing.T, nbData []libovsdb.TestData) (client.Client, *libovsdb.Context) {
	t.Helper()

	nbClient, cleanup, err := libovsdb.NewNBTestHarness(libovsdb.TestSetup{
		NBData: nbData,
	}, nil)
	require.NoError(t, err)

	// Helper function to update logical router port status based on gateway chassis priorities
	updateLogicalRouterPortStatus := func(ctx context.Context, lrp *nbdb.LogicalRouterPort) {
		var selected *nbdb.GatewayChassis
		for _, gcUUID := range lrp.GatewayChassis {
			gc := nbdb.GatewayChassis{UUID: gcUUID}
			err := nbClient.Get(ctx, &gc)
			require.NoError(t, err)

			if selected == nil || gc.Priority > selected.Priority {
				selected = &gc
			}
		}

		agentName := ""
		if selected != nil {
			agentName = selected.ChassisName
		}

		lrp.Status = map[string]string{
			"hosting-chassis": agentName,
		}

		ops, err := nbClient.Where(lrp).Update(lrp)
		require.NoError(t, err)

		_, err = nbClient.Transact(ctx, ops...)
		require.NoError(t, err)
	}

	// NOTE(mnaser): The following code simulates the `status` column which
	//               was added in v23.09 that includes specifically the
	//               `hosting-chassis` field.
	nbClient.Cache().AddEventHandler(&cache.EventHandlerFuncs{
		AddFunc: func(table string, model model.Model) {
			if table == nbdb.LogicalRouterPortTable {
				lrp := model.(*nbdb.LogicalRouterPort)

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				updateLogicalRouterPortStatus(ctx, lrp)
			}
		},
		UpdateFunc: func(table string, oldModel, newModel model.Model) {
			if table == nbdb.GatewayChassisTable {
				gc := newModel.(*nbdb.GatewayChassis)

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				var lrps []nbdb.LogicalRouterPort
				err := nbClient.WhereCache(func(lrp *nbdb.LogicalRouterPort) bool {
					for _, gcUUID := range lrp.GatewayChassis {
						if gcUUID == gc.UUID {
							return true
						}
					}
					return false
				}).List(ctx, &lrps)
				require.NoError(t, err)

				for _, lrp := range lrps {
					updateLogicalRouterPortStatus(ctx, &lrp)
				}
			}
		},
	})

	return nbClient, cleanup
}

func TestList(t *testing.T) {
	tests := []struct {
		name     string
		nbData   []libovsdb.TestData
		expected []types.UID
	}{
		{
			name: "single router",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name: "neutron-" + testRouterUUID,
				},
			},
			expected: []types.UID{testRouterUUID},
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
			expected: []types.UID{testRouterUUID, testRouterUUID2},
		},
		{
			name:     "no routers",
			nbData:   []libovsdb.TestData{},
			expected: []types.UID{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			nbClient, cleanup := setupTestHarnessForTest(t, tt.nbData)
			t.Cleanup(cleanup.Cleanup)

			list, err := List(ctx, nbClient)
			require.NoError(t, err)

			require.Len(t, list.Items, len(tt.expected))
			for _, router := range list.Items {
				assert.Contains(t, tt.expected, router.UID)
			}
		})
	}
}

func TestRouter_LogicalRouterPorts(t *testing.T) {
	tests := []struct {
		name     string
		nbData   []libovsdb.TestData
		expected []string
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
			expected: []string{testPortUUID1, testPortUUID2},
		},
		{
			name: "no ports",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{},
				},
			},
			expected: []string{},
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
			expected: []string{testPortUUID1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			nbClient, cleanup := setupTestHarnessForTest(t, tt.nbData)
			t.Cleanup(cleanup.Cleanup)

			_, lr, err := GetByName(ctx, nbClient, "neutron-"+testRouterUUID)
			require.NoError(t, err)

			ports, err := GetLogicalRouterPorts(ctx, nbClient, lr)
			require.NoError(t, err)

			require.Len(t, ports, len(tt.expected))
			for _, port := range ports {
				assert.Contains(t, tt.expected, port.UUID)
			}
		})
	}
}
func TestRouter_GatewayChassis(t *testing.T) {
	tests := []struct {
		name     string
		nbData   []libovsdb.TestData
		expected []string
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
					UUID:        testChassisUUID,
					Name:        "lrp-1_gwc-1",
					ChassisName: "gwc-1",
					Priority:    1,
				},
			},
			expected: []string{testChassisUUID},
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
					UUID:        testChassisUUID,
					Name:        "lrp-1_gwc-1",
					ChassisName: "gwc-1",
					Priority:    1,
				},
				&nbdb.GatewayChassis{
					UUID:        testChassisUUID2,
					Name:        "lrp-1_gwc-2",
					ChassisName: "gwc-2",
					Priority:    2,
				},
			},
			expected: []string{testChassisUUID, testChassisUUID2},
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
					UUID:        testChassisUUID,
					Name:        "lrp-1_gwc-1",
					ChassisName: "gwc-1",
					Priority:    1,
				},
				&nbdb.GatewayChassis{
					UUID:        testChassisUUID2,
					Name:        "lrp-1_gwc-2",
					ChassisName: "gwc-2",
					Priority:    2,
				},
			},
			expected: []string{testChassisUUID, testChassisUUID2},
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
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			nbClient, cleanup := setupTestHarnessForTest(t, tt.nbData)
			t.Cleanup(cleanup.Cleanup)

			_, lr, err := GetByName(ctx, nbClient, "neutron-"+testRouterUUID)
			require.NoError(t, err)

			gcs, err := GetGatewayChassis(ctx, nbClient, lr)
			require.NoError(t, err)

			require.Len(t, gcs, len(tt.expected))
			for _, gc := range gcs {
				assert.Contains(t, tt.expected, gc.UUID)
			}
		})
	}
}

func TestRouter_HostingAgent(t *testing.T) {
	tests := []struct {
		name          string
		nbData        []libovsdb.TestData
		expectedAgent string
		expectError   bool
		errorContains string
	}{
		{
			name: "single agent hosting gateway port",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{testPortUUID1, testPortUUID2},
				},
				&nbdb.LogicalRouterPort{
					UUID:           testPortUUID1,
					Name:           "lrp-1",
					GatewayChassis: []string{testChassisUUID},
				},
				&nbdb.LogicalRouterPort{
					UUID:           testPortUUID2,
					Name:           "lrp-2",
					GatewayChassis: []string{},
				},
				&nbdb.GatewayChassis{
					UUID:        testChassisUUID,
					Name:        "lrp-1_gwc-1",
					ChassisName: "gwc-1",
					Priority:    1,
				},
			},
			expectedAgent: "gwc-1",
		},
		{
			name: "multiple agents hosting gateway port",
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
					UUID:        testChassisUUID,
					Name:        "lrp-1_gwc-1",
					ChassisName: "gwc-1",
					Priority:    1,
				},
				&nbdb.GatewayChassis{
					UUID:        testChassisUUID2,
					Name:        "lrp-1_gwc-2",
					ChassisName: "gwc-2",
					Priority:    2,
				},
			},
			expectedAgent: "gwc-2",
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
			errorContains: "no hosting-chassis found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			nbClient, cleanup := setupTestHarnessForTest(t, tt.nbData)
			t.Cleanup(cleanup.Cleanup)

			_, lr, err := GetByName(ctx, nbClient, "neutron-"+testRouterUUID)
			require.NoError(t, err)

			// NOTE(mnaser): I hate this, but this gives a chance to the handlers to
			//               reconcile and update the status field.
			time.Sleep(10 * time.Millisecond)

			agent, err := GetHostingAgent(ctx, nbClient, lr)

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

func TestRouter_Failover(t *testing.T) {
	tests := []struct {
		name                  string
		nbData                []libovsdb.TestData
		expectedInitialAgent  string
		expectedFailoverAgent string
		expectError           bool
		errorContains         string
	}{
		{
			name: "no gateway chassis",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{testPortUUID1, testPortUUID2},
				},
				&nbdb.LogicalRouterPort{
					UUID:           testPortUUID1,
					Name:           "lrp-1",
					GatewayChassis: []string{},
				},
				&nbdb.LogicalRouterPort{
					UUID:           testPortUUID2,
					Name:           "lrp-2",
					GatewayChassis: []string{},
				},
			},
			expectError:   true,
			errorContains: "no gateway chassis found",
		},
		{
			name: "single gateway chassis",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{testPortUUID1, testPortUUID2},
				},
				&nbdb.LogicalRouterPort{
					UUID:           testPortUUID1,
					Name:           "lrp-1",
					GatewayChassis: []string{testChassisUUID},
				},
				&nbdb.LogicalRouterPort{
					UUID:           testPortUUID2,
					Name:           "lrp-2",
					GatewayChassis: []string{},
				},
				&nbdb.GatewayChassis{
					UUID:        testChassisUUID,
					Name:        "lrp-1_gwc-1",
					ChassisName: "gwc-1",
					Priority:    10,
				},
			},
			expectedInitialAgent: "gwc-1",
			expectError:          true,
			errorContains:        "only one gateway chassis",
		},
		{
			name: "dual gateway chassis",
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
					UUID:        testChassisUUID,
					Name:        "lrp-1_gwc-1",
					ChassisName: "gwc-1",
					Priority:    1,
				},
				&nbdb.GatewayChassis{
					UUID:        testChassisUUID2,
					Name:        "lrp-1_gwc-2",
					ChassisName: "gwc-2",
					Priority:    2,
				},
			},
			expectedInitialAgent:  "gwc-2",
			expectedFailoverAgent: "gwc-1",
		},
		{
			name: "triple gateway chassis",
			nbData: []libovsdb.TestData{
				&nbdb.LogicalRouter{
					Name:  "neutron-" + testRouterUUID,
					Ports: []string{testPortUUID1, testPortUUID2},
				},
				&nbdb.LogicalRouterPort{
					UUID:           testPortUUID1,
					Name:           "lrp-1",
					GatewayChassis: []string{testChassisUUID, testChassisUUID2, testChassisUUID3},
				},
				&nbdb.LogicalRouterPort{
					UUID:           testPortUUID2,
					Name:           "lrp-2",
					GatewayChassis: []string{},
				},
				&nbdb.GatewayChassis{
					UUID:        testChassisUUID,
					Name:        "lrp-1_gwc-1",
					ChassisName: "gwc-1",
					Priority:    1,
				},
				&nbdb.GatewayChassis{
					UUID:        testChassisUUID2,
					Name:        "lrp-1_gwc-2",
					ChassisName: "gwc-2",
					Priority:    2,
				},
				&nbdb.GatewayChassis{
					UUID:        testChassisUUID3,
					Name:        "lrp-1_gwc-3",
					ChassisName: "gwc-3",
					Priority:    3,
				},
			},
			expectedInitialAgent:  "gwc-3",
			expectedFailoverAgent: "gwc-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			nbClient, cleanup := setupTestHarnessForTest(t, tt.nbData)
			t.Cleanup(cleanup.Cleanup)

			router, lr, err := GetByName(ctx, nbClient, "neutron-"+testRouterUUID)
			require.NoError(t, err)

			// NOTE(mnaser): I hate this, but this gives a chance to the handlers to
			//               reconcile and update the status field.
			time.Sleep(10 * time.Millisecond)

			if tt.expectedInitialAgent != "" {
				agent, err := GetHostingAgent(ctx, nbClient, lr)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedInitialAgent, agent)
			}

			err = Failover(ctx, nbClient, router)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)

				if tt.expectedFailoverAgent != "" {
					agent, err := GetHostingAgent(ctx, nbClient, lr)
					require.NoError(t, err)
					assert.Equal(t, tt.expectedFailoverAgent, agent)
				}
			}
		})
	}
}
