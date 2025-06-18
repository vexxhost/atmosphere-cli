//go:build integration
// +build integration

package components

import (
	"context"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/spf13/viper"
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
	configFlags   *genericclioptions.ConfigFlags
	ctx           context.Context
	metricsServer *MetricsServer
	client        *helm.Client
	chartConfig   *helm.ChartConfig
	releaseConfig *helm.ReleaseConfig
}

func (suite *MetricsServerTestSuite) SetupTest() {
	viper.Reset()

	suite.configFlags = genericclioptions.NewConfigFlags(true)
	suite.ctx = atmosphere.New(context.Background(), suite.configFlags)
	suite.metricsServer = NewMetricsServer()
	suite.chartConfig = suite.metricsServer.GetChartConfig(suite.ctx)
	suite.releaseConfig = suite.metricsServer.GetReleaseConfig(suite.ctx)

	// Create client
	var err error
	suite.client, err = helm.NewClient(suite.configFlags, suite.releaseConfig.Namespace)
	require.NoError(suite.T(), err)

	// Ensure clean state
	exists, err := suite.client.ReleaseExists(suite.releaseConfig.Name)
	require.NoError(suite.T(), err)

	if exists {
		suite.T().Log("Found existing metrics-server, uninstalling for clean test...")
		err = suite.client.UninstallRelease(suite.releaseConfig.Name)
		require.NoError(suite.T(), err)
		time.Sleep(10 * time.Second)
	}
}

func (suite *MetricsServerTestSuite) TearDownTest() {
	suite.T().Log("Cleaning up metrics-server installation...")
	err := suite.client.UninstallRelease(suite.releaseConfig.Name)
	if err != nil {
		suite.T().Logf("Failed to uninstall metrics-server during cleanup: %v", err)
	}
}

func (suite *MetricsServerTestSuite) TestDeployMetricsServerAndVerifyAPI() {
	ctx := context.Background()

	_, err := suite.client.DeployRelease(suite.chartConfig, suite.releaseConfig)
	require.NoError(suite.T(), err)

	clientConfig, err := suite.configFlags.ToRESTConfig()
	require.NoError(suite.T(), err)

	metricsClient, err := metricsclientset.NewForConfig(clientConfig)
	require.NoError(suite.T(), err)

	err = retry.Do(
		func() error {
			_, err := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
			return err
		},
		retry.Attempts(24),
		retry.Delay(5*time.Second),
		retry.LastErrorOnly(true),
	)
	require.NoError(suite.T(), err, "Metrics API did not become available")

	nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), nodeMetrics.Items, "No node metrics available")

	podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses("kube-system").List(ctx, metav1.ListOptions{})
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), podMetrics.Items, "No pod metrics available")

	suite.T().Logf("Metrics API is working: %d nodes and %d pods reporting metrics", 
		len(nodeMetrics.Items), len(podMetrics.Items))
}

func (suite *MetricsServerTestSuite) TestRedeploymentDoesNotChangeRevision() {
	_, err := suite.client.DeployRelease(suite.chartConfig, suite.releaseConfig)
	require.NoError(suite.T(), err)

	deployedRelease, err := suite.client.GetRelease(suite.releaseConfig.Name)
	require.NoError(suite.T(), err)
	initialRevision := deployedRelease.Version

	_, err = suite.client.DeployRelease(suite.chartConfig, suite.releaseConfig)
	require.NoError(suite.T(), err)

	deployedRelease, err = suite.client.GetRelease(suite.releaseConfig.Name)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), initialRevision, deployedRelease.Version, 
		"Revision should not change when there are no configuration changes")
}

func (suite *MetricsServerTestSuite) TestCustomConfiguration() {
	viper.Set("metrics-server.chart.version", "3.12.1")
	
	customChartConfig := suite.metricsServer.GetChartConfig(suite.ctx)
	assert.Equal(suite.T(), "3.12.1", customChartConfig.Version)
}

func TestMetricsServerSuite(t *testing.T) {
	suite.Run(t, &MetricsServerTestSuite{})
}
