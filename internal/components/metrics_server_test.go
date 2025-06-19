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
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

type MetricsServerTestSuite struct {
	helm.HelmTestSuite
}

// SetupSuite initializes common test infrastructure
func (suite *MetricsServerTestSuite) SetupSuite() {
	// Call parent setup
	suite.HelmTestSuite.SetupSuite()
	// Override context with atmosphere context
	suite.Ctx = atmosphere.New(suite.Ctx, suite.ConfigFlags)
}

func (suite *MetricsServerTestSuite) TestDeployment() {
	metricsServer := NewMetricsServer(nil)
	componentConfig, err := metricsServer.MergedConfig()
	require.NoError(suite.T(), err)

	// Deploy the component
	_, err = suite.DeployComponent(componentConfig)
	require.NoError(suite.T(), err)

	clientConfig, err := suite.ConfigFlags.ToRESTConfig()
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

	// Deploy the component
	deployedRelease, err := suite.DeployComponent(componentConfig)
	require.NoError(suite.T(), err)

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
