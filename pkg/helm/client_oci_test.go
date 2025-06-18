//go:build integration
// +build integration

package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// OCIChartTestSuite is a sub-suite for OCI chart deployment tests
type OCIChartTestSuite struct {
	BaseClientTestSuite
}

func (suite *OCIChartTestSuite) TestDeployMemcached() {
	// Configure test release with memcached OCI chart
	chartConfig := &ChartConfig{
		Name: "oci://registry-1.docker.io/bitnamicharts/memcached:7.8.6",
	}
	releaseConfig := &ReleaseConfig{
		Namespace: "test-helm-oci",
		Name:      "test-memcached",
		Values:    map[string]interface{}{},
	}

	// Create client
	client, err := NewClient(suite.configFlags, releaseConfig.Namespace)
	require.NoError(suite.T(), err)

	// Prepare release (ensures clean state and tracks for cleanup)
	suite.PrepareRelease(client, releaseConfig)

	// Deploy the chart
	_, err = client.DeployRelease(chartConfig, releaseConfig)
	require.NoError(suite.T(), err)

	// Verify release exists
	exists, err := client.ReleaseExists(releaseConfig.Name)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), exists, "Release should exist after deployment")

	// Get deployed release info
	deployedRelease, err := client.GetRelease(releaseConfig.Name)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "test-memcached", deployedRelease.Name)
	assert.Equal(suite.T(), "test-helm-oci", deployedRelease.Namespace)
	assert.Equal(suite.T(), 1, deployedRelease.Version)

	suite.T().Logf("Successfully deployed OCI chart: %s in namespace %s",
		deployedRelease.Name, deployedRelease.Namespace)
}

func TestOCIChartSuite(t *testing.T) {
	suite.Run(t, &OCIChartTestSuite{})
}
