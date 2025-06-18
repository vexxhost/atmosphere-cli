//go:build integration

package helm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// BaseClientTestSuite provides common functionality for all client test suites
type BaseClientTestSuite struct {
	suite.Suite
	configFlags *genericclioptions.ConfigFlags
	ctx         context.Context
	// Track releases for cleanup
	releases []struct {
		client        *Client
		releaseConfig *ReleaseConfig
	}
}

// SetupSuite initializes common test infrastructure
func (suite *BaseClientTestSuite) SetupSuite() {
	suite.configFlags = genericclioptions.NewConfigFlags(true)
	suite.ctx = context.Background()
	suite.releases = make([]struct {
		client        *Client
		releaseConfig *ReleaseConfig
	}, 0)
}

// TearDownSuite cleans up all tracked releases
func (suite *BaseClientTestSuite) TearDownSuite() {
	for _, r := range suite.releases {
		if r.client != nil && r.releaseConfig != nil {
			exists, err := r.client.ReleaseExists(r.releaseConfig.Name)
			if err == nil && exists {
				suite.T().Logf("Cleaning up release: %s in namespace %s",
					r.releaseConfig.Name, r.releaseConfig.Namespace)
				_ = r.client.UninstallRelease(r.releaseConfig.Name)
			}
		}
	}
}

// EnsureCleanState removes any existing release before test
func (suite *BaseClientTestSuite) EnsureCleanState(client *Client, releaseConfig *ReleaseConfig) {
	exists, err := client.ReleaseExists(releaseConfig.Name)
	require.NoError(suite.T(), err)

	if exists {
		suite.T().Logf("Found existing %s, uninstalling for clean test...", releaseConfig.Name)
		err = client.UninstallRelease(releaseConfig.Name)
		require.NoError(suite.T(), err)
		time.Sleep(5 * time.Second)
	}
}

// TrackRelease registers a release for automatic cleanup
func (suite *BaseClientTestSuite) TrackRelease(client *Client, releaseConfig *ReleaseConfig) {
	suite.releases = append(suite.releases, struct {
		client        *Client
		releaseConfig *ReleaseConfig
	}{client: client, releaseConfig: releaseConfig})
}

// PrepareRelease ensures clean state and tracks the release for cleanup
func (suite *BaseClientTestSuite) PrepareRelease(client *Client, releaseConfig *ReleaseConfig) {
	suite.TrackRelease(client, releaseConfig)
	suite.EnsureCleanState(client, releaseConfig)
}

type ClientTestSuite struct {
	suite.Suite
	configFlags *genericclioptions.ConfigFlags
	ctx         context.Context
}

func (suite *ClientTestSuite) SetupTest() {
	suite.configFlags = genericclioptions.NewConfigFlags(true)
	suite.ctx = context.Background()
}

func (suite *ClientTestSuite) TearDownTest() {
	// Cleanup will be done in individual tests if needed
}

func TestClientSuite(t *testing.T) {
	suite.Run(t, &ClientTestSuite{})
}
