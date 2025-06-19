//go:build integration

package helm

import (
	"testing"

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

	// Deploy the component
	_, err := suite.DeployComponent(componentConfig)
	require.NoError(suite.T(), err)
}

func TestRemoteChartSuite(t *testing.T) {
	suite.Run(t, &RemoteChartTestSuite{})
}
