package helm

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Client provides a Helm client for interacting with a Kubernetes cluster
type Client struct {
	getter       genericclioptions.RESTClientGetter
	Namespace    string
	actionConfig *action.Configuration
}

// NewClient creates a new Helm client
func NewClient(getter genericclioptions.RESTClientGetter, namespace string) (*Client, error) {
	registryClient, err := registry.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create registry client: %w", err)
	}

	actionConfig := new(action.Configuration)
	actionConfig.RegistryClient = registryClient

	if err := actionConfig.Init(getter, namespace, os.Getenv("HELM_DRIVER"), func(format string, args ...interface{}) {
		log.With("namespace", namespace).Debugf(format, args...)
	}); err != nil {
		return nil, fmt.Errorf("failed to initialize action config: %w", err)
	}

	if err := actionConfig.KubeClient.IsReachable(); err != nil {
		return nil, fmt.Errorf("kubernetes cluster is not reachable: %w", err)
	}

	return &Client{
		getter:       getter,
		Namespace:    namespace,
		actionConfig: actionConfig,
	}, nil
}

// loadChart loads a Helm chart from the given chart configuration
func (c *Client) loadChart(chartName string, chartPathOptions action.ChartPathOptions) (*chart.Chart, error) {
	chartPath, err := chartPathOptions.LocateChart(chartName, cli.New())
	if err != nil {
		return nil, err
	}

	return loader.Load(chartPath)
}

// configureInstallAction creates and configures an install action
func (c *Client) configureInstallAction(config *ComponentConfig) *action.Install {
	install := action.NewInstall(c.actionConfig)

	install.RepoURL = config.Chart.RepoURL
	install.ReleaseName = config.Release.Name
	install.Version = config.Chart.Version

	install.Namespace = config.Release.Namespace
	install.CreateNamespace = true

	install.Wait = true
	install.Timeout = 5 * time.Minute

	return install
}

// configureUpgradeAction creates and configures an upgrade action
func (c *Client) configureUpgradeAction(config *ComponentConfig, dryRun bool) *action.Upgrade {
	upgrade := action.NewUpgrade(c.actionConfig)

	upgrade.Install = true
	upgrade.RepoURL = config.Chart.RepoURL
	upgrade.Version = config.Chart.Version

	upgrade.Namespace = config.Release.Namespace
	upgrade.ResetValues = true
	upgrade.Wait = true
	upgrade.Timeout = 5 * time.Minute
	upgrade.DryRun = dryRun

	return upgrade
}

// ReleaseExists checks if a release exists
func (c *Client) ReleaseExists(releaseName string) (bool, error) {
	history := action.NewHistory(c.actionConfig)
	history.Max = 1

	_, err := history.Run(releaseName)
	return err == nil, nil
}

// UninstallRelease removes a Helm release
func (c *Client) UninstallRelease(releaseName string) error {
	uninstall := action.NewUninstall(c.actionConfig)
	_, err := uninstall.Run(releaseName)
	if err != nil {
		return fmt.Errorf("failed to uninstall release %s: %w", releaseName, err)
	}

	log.Info("Successfully uninstalled release", "name", releaseName)
	return nil
}

// GetRelease retrieves a deployed release
func (c *Client) GetRelease(releaseName string) (*release.Release, error) {
	get := action.NewGet(c.actionConfig)
	return get.Run(releaseName)
}

// InstallRelease installs a Helm chart as a new release
func (c *Client) InstallRelease(config *ComponentConfig) (*release.Release, error) {
	install := c.configureInstallAction(config)

	ch, err := c.loadChart(config.Chart.Name, install.ChartPathOptions)
	if err != nil {
		return nil, err
	}

	return install.Run(ch, config.Release.Values)
}

// UpgradeRelease upgrades an existing Helm release
func (c *Client) UpgradeRelease(config *ComponentConfig) (*release.Release, error) {
	upgrade := c.configureUpgradeAction(config, false)

	ch, err := c.loadChart(config.Chart.Name, upgrade.ChartPathOptions)
	if err != nil {
		return nil, err
	}

	return upgrade.Run(config.Release.Name, ch, config.Release.Values)
}

// GetTemplatedManifests renders chart templates with the provided values using a dry-run
func (c *Client) GetTemplatedManifests(config *ComponentConfig) (string, error) {
	upgrade := c.configureUpgradeAction(config, true)

	ch, err := c.loadChart(config.Chart.Name, upgrade.ChartPathOptions)
	if err != nil {
		return "", err
	}

	rel, err := upgrade.Run(config.Release.Name, ch, config.Release.Values)
	if err != nil {
		return "", err
	}

	return rel.Manifest, nil
}

// HasDiff checks if there are differences between deployed and templated manifests
func (c *Client) HasDiff(config *ComponentConfig) (bool, error) {
	// Check if release exists
	exists, err := c.ReleaseExists(config.Release.Name)
	if err != nil {
		return false, err
	}

	// If release doesn't exist, there's always a diff (new installation)
	if !exists {
		return true, nil
	}

	// Get deployed release
	deployedRelease, err := c.GetRelease(config.Release.Name)
	if err != nil {
		return false, fmt.Errorf("failed to get deployed release: %w", err)
	}

	// Get templated manifests
	templatedManifests, err := c.GetTemplatedManifests(config)
	if err != nil {
		return false, fmt.Errorf("failed to get templated manifests: %w", err)
	}

	// Compare manifests
	return deployedRelease.Manifest != templatedManifests, nil
}

// DeployRelease installs or upgrades a release based on whether it exists
func (c *Client) DeployRelease(config *ComponentConfig) (*release.Release, error) {
	exists, err := c.ReleaseExists(config.Release.Name)
	if err != nil {
		return nil, err
	}

	if exists {
		// Check if there are any changes
		hasDiff, err := c.HasDiff(config)
		if err != nil {
			return nil, fmt.Errorf("failed to check for differences: %w", err)
		}

		// If no differences, skip upgrade
		if !hasDiff {
			log.Info("No changes detected, skipping upgrade", "name", config.Release.Name)
			// Get current release
			return c.GetRelease(config.Release.Name)
		}

		return c.UpgradeRelease(config)
	}

	return c.InstallRelease(config)
}
