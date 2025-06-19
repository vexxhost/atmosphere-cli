//go:build integration

package helm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// ReleaseTracker tracks releases for cleanup
type ReleaseTracker struct {
	releases []struct {
		client        *Client
		releaseConfig *ReleaseConfig
	}
}

// NewReleaseTracker creates a new release tracker
func NewReleaseTracker() *ReleaseTracker {
	return &ReleaseTracker{
		releases: make([]struct {
			client        *Client
			releaseConfig *ReleaseConfig
		}, 0),
	}
}

// Track registers a release for automatic cleanup
func (rt *ReleaseTracker) Track(client *Client, releaseConfig *ReleaseConfig) {
	rt.releases = append(rt.releases, struct {
		client        *Client
		releaseConfig *ReleaseConfig
	}{client: client, releaseConfig: releaseConfig})
}

// CleanupAll removes all tracked releases
func (rt *ReleaseTracker) CleanupAll(t *testing.T) {
	for _, r := range rt.releases {
		if r.client != nil && r.releaseConfig != nil {
			exists, err := r.client.ReleaseExists(r.releaseConfig.Name)
			if err == nil && exists {
				t.Logf("Cleaning up release: %s in namespace %s",
					r.releaseConfig.Name, r.releaseConfig.Namespace)
				_ = r.client.UninstallRelease(r.releaseConfig.Name)
			}
		}
	}
}

// HelmTestSuite provides a base test suite for Helm-based tests
type HelmTestSuite struct {
	suite.Suite
	ConfigFlags *genericclioptions.ConfigFlags
	Ctx         context.Context
	Tracker     *ReleaseTracker
}

// SetupSuite initializes the test suite
func (s *HelmTestSuite) SetupSuite() {
	s.ConfigFlags = genericclioptions.NewConfigFlags(true)
	s.Ctx = context.Background()
	s.Tracker = NewReleaseTracker()
}

// TearDownSuite cleans up all tracked releases
func (s *HelmTestSuite) TearDownSuite() {
	s.Tracker.CleanupAll(s.T())
}

// PrepareRelease prepares a release for testing by ensuring clean state and tracking for cleanup
func (s *HelmTestSuite) PrepareRelease(client *Client, releaseConfig *ReleaseConfig) {
	s.Tracker.Track(client, releaseConfig)

	// Ensure clean state
	exists, err := client.ReleaseExists(releaseConfig.Name)
	require.NoError(s.T(), err)

	if exists {
		s.T().Logf("Found existing %s, uninstalling for clean test...", releaseConfig.Name)
		err = client.UninstallRelease(releaseConfig.Name)
		require.NoError(s.T(), err)
		time.Sleep(5 * time.Second)
	}
}

// CreateClient creates a new Helm client for the given namespace
func (s *HelmTestSuite) CreateClient(namespace string) (*Client, error) {
	return NewClient(s.ConfigFlags, namespace)
}

func (s *HelmTestSuite) DeployComponent(componentConfig *ComponentConfig) (*release.Release, error) {
	// Create client
	client, err := s.CreateClient(componentConfig.Release.Namespace)
	if err != nil {
		return nil, err
	}

	// Prepare release (ensures clean state and tracks for cleanup)
	s.PrepareRelease(client, componentConfig.Release)

	// Deploy the release
	_, err = client.DeployRelease(componentConfig)
	if err != nil {
		return nil, err
	}

	// Verify release exists
	exists, err := client.ReleaseExists(componentConfig.Release.Name)
	require.NoError(s.T(), err)
	require.True(s.T(), exists, "Release %s should exist after deployment", componentConfig.Release.Name)

	// Get deployed release info
	deployedRelease, err := client.GetRelease(componentConfig.Release.Name)
	require.NoError(s.T(), err)

	// Verify deployment status and basic info
	require.Equal(s.T(), "deployed", deployedRelease.Info.Status.String())
	require.Equal(s.T(), componentConfig.Release.Name, deployedRelease.Name)
	require.Equal(s.T(), componentConfig.Release.Namespace, deployedRelease.Namespace)
	require.Equal(s.T(), 1, deployedRelease.Version)

	// Log success
	s.T().Logf("Successfully deployed %s: %s in namespace %s",
		componentConfig.Chart.Name, deployedRelease.Name, deployedRelease.Namespace)

	return deployedRelease, nil
}
