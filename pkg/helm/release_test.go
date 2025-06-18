//go:build integration
// +build integration

package helm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// BaseReleaseTestSuite provides common functionality for all release test suites
type BaseReleaseTestSuite struct {
	suite.Suite
	configFlags *genericclioptions.ConfigFlags
	ctx         context.Context
	releases    []*Release
}

// SetupSuite initializes common test infrastructure
func (suite *BaseReleaseTestSuite) SetupSuite() {
	suite.configFlags = genericclioptions.NewConfigFlags(true)
	suite.ctx = context.Background()
	suite.releases = make([]*Release, 0)
}

// TearDownSuite cleans up all tracked releases
func (suite *BaseReleaseTestSuite) TearDownSuite() {
	for _, release := range suite.releases {
		if release != nil {
			exists, err := release.Exists()
			if err == nil && exists {
				suite.T().Logf("Cleaning up release: %s in namespace %s",
					release.ReleaseConfig.Name, release.ReleaseConfig.Namespace)
				_ = release.Uninstall()
			}
		}
	}
}

// EnsureCleanState removes any existing release before test
func (suite *BaseReleaseTestSuite) EnsureCleanState(release *Release) {
	exists, err := release.Exists()
	require.NoError(suite.T(), err)

	if exists {
		suite.T().Logf("Found existing %s, uninstalling for clean test...", release.ReleaseConfig.Name)
		err = release.Uninstall()
		require.NoError(suite.T(), err)
		time.Sleep(5 * time.Second)
	}
}

// TrackRelease registers a release for automatic cleanup
func (suite *BaseReleaseTestSuite) TrackRelease(release *Release) {
	suite.releases = append(suite.releases, release)
}

// PrepareRelease ensures clean state and tracks the release for cleanup
func (suite *BaseReleaseTestSuite) PrepareRelease(release *Release) {
	suite.TrackRelease(release)
	suite.EnsureCleanState(release)
}

type ReleaseTestSuite struct {
	suite.Suite
	configFlags *genericclioptions.ConfigFlags
	ctx         context.Context
}

func (suite *ReleaseTestSuite) SetupTest() {
	suite.configFlags = genericclioptions.NewConfigFlags(true)
	suite.ctx = context.Background()
}

func (suite *ReleaseTestSuite) TearDownTest() {
	// Cleanup will be done in individual tests if needed
}

func TestReleaseSuite(t *testing.T) {
	suite.Run(t, &ReleaseTestSuite{})
}
