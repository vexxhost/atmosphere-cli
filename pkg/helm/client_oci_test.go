//go:build integration

package helm

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// OCIChartTestSuite is a sub-suite for OCI chart deployment tests
type OCIChartTestSuite struct {
	HelmTestSuite
}

func (suite *OCIChartTestSuite) TestDeployMemcached() {
	// Configure test component with memcached OCI chart
	componentConfig := &ComponentConfig{
		Chart: &ChartConfig{
			Name: "oci://registry-1.docker.io/bitnamicharts/memcached:7.8.6",
		},
		Release: &ReleaseConfig{
			Namespace: "test-helm-oci",
			Name:      "test-memcached",
			Values:    map[string]interface{}{},
		},
	}

	// Deploy the component
	_, err := suite.DeployComponent(componentConfig)
	require.NoError(suite.T(), err)
}

func TestOCIChartSuite(t *testing.T) {
	suite.Run(t, &OCIChartTestSuite{})
}
