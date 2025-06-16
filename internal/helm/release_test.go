//go:build integration
// +build integration

package helm

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vexxhost/atmosphere/internal/config"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ReleaseTestSuite struct {
	suite.Suite
	configFlags *genericclioptions.ConfigFlags
	ctx         context.Context
}

func (suite *ReleaseTestSuite) SetupTest() {
	viper.Reset()

	suite.configFlags = genericclioptions.NewConfigFlags(true)
	suite.ctx = context.Background()
}

func (suite *ReleaseTestSuite) TearDownTest() {
	// Cleanup will be done in individual tests if needed
}

func (suite *ReleaseTestSuite) TestDeployOCIChart() {
	// Configure test release with memcached OCI chart
	release := &Release{
		RESTClientGetter: suite.configFlags,
		ChartConfig: &config.ChartConfig{
			Name:    "oci://registry-1.docker.io/bitnamicharts/memcached:7.8.6",
		},
		ReleaseConfig: &config.ReleaseConfig{
			Namespace: "test-helm-oci",
			Name:      "test-memcached",
			Values:    map[string]interface{}{},
		},
	}

	// Ensure clean state
	exists, err := release.Exists()
	require.NoError(suite.T(), err)

	if exists {
		suite.T().Log("Found existing test-memcached, uninstalling for clean test...")
		err = release.Uninstall()
		require.NoError(suite.T(), err)
		time.Sleep(5 * time.Second)
	}

	// Deploy the chart
	err = release.Deploy()
	require.NoError(suite.T(), err)
	defer release.Uninstall()

	// Verify release exists
	exists, err = release.Exists()
	require.NoError(suite.T(), err)
	assert.True(suite.T(), exists, "Release should exist after deployment")

	// Get deployed release info
	deployedRelease, err := release.GetDeployedRelease()
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "test-memcached", deployedRelease.Name)
	assert.Equal(suite.T(), "test-helm-oci", deployedRelease.Namespace)
	assert.Equal(suite.T(), 1, deployedRelease.Version)

	suite.T().Logf("Successfully deployed OCI chart: %s in namespace %s",
		deployedRelease.Name, deployedRelease.Namespace)
}


func TestReleaseSuite(t *testing.T) {
	suite.Run(t, &ReleaseTestSuite{})
}
