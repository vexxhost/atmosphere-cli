//go:build integration

package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// RemoteChartTestSuite is a suite for testing charts from remote repositories with index.yaml
type RemoteChartTestSuite struct {
	HelmTestSuite
}

func (suite *RemoteChartTestSuite) TestDeployFromRemoteRepository() {
	// Configure test component
	componentConfig := &ComponentConfig{
		Chart: &ChartConfig{
			RepoURL: "https://charts.bitnami.com/bitnami",
			Name:    "nginx",
			Version: "18.2.5", // Specify a version for consistency
		},
		Release: &ReleaseConfig{
			Namespace: "test-helm-remote",
			Name:      "test-nginx-remote",
			Values: map[string]interface{}{
				"service": map[string]interface{}{
					"type": "ClusterIP",
				},
			},
		},
	}

	// Create client
	client, err := suite.CreateClient(componentConfig.Release.Namespace)
	require.NoError(suite.T(), err)

	// Prepare release (ensures clean state and tracks for cleanup)
	suite.PrepareRelease(client, componentConfig.Release)

	// Deploy the chart
	_, err = client.DeployRelease(componentConfig)
	require.NoError(suite.T(), err)

	// Verify release exists
	exists, err := client.ReleaseExists(componentConfig.Release.Name)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), exists, "Release should exist after deployment")

	// Get deployed release info
	deployedRelease, err := client.GetRelease(componentConfig.Release.Name)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "test-nginx-remote", deployedRelease.Name)
	assert.Equal(suite.T(), "test-helm-remote", deployedRelease.Namespace)
	assert.Equal(suite.T(), 1, deployedRelease.Version)

	suite.T().Logf("Successfully deployed chart from remote repository: %s in namespace %s",
		deployedRelease.Name, deployedRelease.Namespace)
}

func TestRemoteChartSuite(t *testing.T) {
	suite.Run(t, &RemoteChartTestSuite{})
}
