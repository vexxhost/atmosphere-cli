//go:build integration

package components

import (
	"context"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vexxhost/atmosphere/internal/atmosphere"
	"github.com/vexxhost/atmosphere/pkg/helm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

type MetricsServerTestSuite struct {
	suite.Suite
	configFlags *genericclioptions.ConfigFlags
	ctx         context.Context
	client      *helm.Client
}

func (suite *MetricsServerTestSuite) SetupTest() {
	suite.configFlags = genericclioptions.NewConfigFlags(true)
	suite.ctx = atmosphere.New(context.Background(), suite.configFlags)

	// Create client
	var err error
	suite.client, err = helm.NewClient(suite.configFlags, "kube-system")
	require.NoError(suite.T(), err)

	// Ensure clean state
	exists, err := suite.client.ReleaseExists("metrics-server")
	require.NoError(suite.T(), err)

	if exists {
		suite.T().Log("Found existing metrics-server, uninstalling for clean test...")
		err = suite.client.UninstallRelease("metrics-server")
		require.NoError(suite.T(), err)
		time.Sleep(10 * time.Second)
	}
}

func (suite *MetricsServerTestSuite) TearDownTest() {
	suite.T().Log("Cleaning up metrics-server installation...")
	err := suite.client.UninstallRelease("metrics-server")
	if err != nil {
		suite.T().Logf("Failed to uninstall metrics-server during cleanup: %v", err)
	}
}

func (suite *MetricsServerTestSuite) TestDeployment() {
	metricsServer := NewMetricsServer(nil)
	componentConfig, err := metricsServer.MergedConfig()
	require.NoError(suite.T(), err)

	_, err = suite.client.DeployRelease(componentConfig)
	require.NoError(suite.T(), err)

	clientConfig, err := suite.configFlags.ToRESTConfig()
	require.NoError(suite.T(), err)

	metricsClient, err := metricsclientset.NewForConfig(clientConfig)
	require.NoError(suite.T(), err)

	err = retry.Do(
		func() error {
			_, err := metricsClient.MetricsV1beta1().NodeMetricses().List(context.Background(), metav1.ListOptions{})
			return err
		},
		retry.Attempts(24),
		retry.Delay(5*time.Second),
		retry.LastErrorOnly(true),
	)
	require.NoError(suite.T(), err, "Metrics API did not become available")

	nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().List(context.Background(), metav1.ListOptions{})
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), nodeMetrics.Items, "No node metrics available")
}

func (suite *MetricsServerTestSuite) TestDeploymentWithOverrides() {
	overrides := &helm.ComponentConfig{
		Release: &helm.ReleaseConfig{
			Values: map[string]interface{}{
				"replicas": 2,
				"args": []string{
					"--kubelet-insecure-tls",
					"--metric-resolution=30s",
				},
			},
		},
	}

	metricsServer := NewMetricsServer(overrides)
	componentConfig, err := metricsServer.MergedConfig()
	require.NoError(suite.T(), err)

	_, err = suite.client.DeployRelease(componentConfig)
	require.NoError(suite.T(), err)

	deployedRelease, err := suite.client.GetRelease("metrics-server")
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "deployed", deployedRelease.Info.Status.String())

	// Verify that overrides are properly applied
	// deployedRelease.Config is a map[string]interface{} from helm
	valuesMap := deployedRelease.Config
	suite.T().Logf("Deployed values: %v", valuesMap)

	// Check that replicas was overridden
	replicas, ok := valuesMap["replicas"].(int)
	if !ok {
		// Try float64 as JSON unmarshaling often uses float64 for numbers
		replicasFloat, ok := valuesMap["replicas"].(float64)
		require.True(suite.T(), ok, "replicas should be a number")
		replicas = int(replicasFloat)
	}
	assert.Equal(suite.T(), 2, replicas)

	// Check that args contains both default and override values
	args, ok := valuesMap["args"].([]interface{})
	require.True(suite.T(), ok, "args should be a slice")
	assert.Contains(suite.T(), args, "--kubelet-insecure-tls")
	assert.Contains(suite.T(), args, "--metric-resolution=30s")
}

func TestMetricsServerSuite(t *testing.T) {
	suite.Run(t, &MetricsServerTestSuite{})
}
