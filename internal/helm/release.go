package helm

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
	"github.com/vexxhost/atmosphere/internal/config"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func SetReleaseDefault(section *viper.Viper, defaults *Release) {
	section.SetDefault("chart.repository", defaults.ChartConfig.RepoURL)
	section.SetDefault("chart.name", defaults.ChartConfig.Name)
	section.SetDefault("chart.version", defaults.ChartConfig.Version)
	section.SetDefault("release.namespace", defaults.ReleaseConfig.Namespace)
	section.SetDefault("release.name", defaults.ReleaseConfig.Name)
	section.SetDefault("release.values", defaults.ReleaseConfig.Values)
}

// Release represents a Helm release manager that handles install/upgrade operations
type Release struct {
	RESTClientGetter genericclioptions.RESTClientGetter
	ChartConfig      *config.ChartConfig
	ReleaseConfig    *config.ReleaseConfig
	Revision         int
}

func (r *Release) GetActionConfig() (*action.Configuration, error) {
	registryClient, err := registry.NewClient()
	if err != nil {
		return nil, err
	}

	actionConfig := new(action.Configuration)
	actionConfig.RegistryClient = registryClient

	if err := actionConfig.Init(r.RESTClientGetter, r.ReleaseConfig.Namespace, os.Getenv("HELM_DRIVER"), func(format string, args ...interface{}) {
		log.With("namespace", r.ReleaseConfig.Namespace).With("release", r.ReleaseConfig.Name).With("version", r.ChartConfig.Version).Debugf(format, args...)
	}); err != nil {
		log.Fatal("Failed to initialize Helm action config", "error", err)
	}

	if err := actionConfig.KubeClient.IsReachable(); err != nil {
		return nil, err
	}

	return actionConfig, nil
}

// GetChart retrieves the Helm chart based on the provided ChartPathOptions
func (r *Release) GetChart(chartPathOptions action.ChartPathOptions) (*chart.Chart, error) {
	chartPath, err := chartPathOptions.LocateChart(r.ChartConfig.Name, cli.New())
	if err != nil {
		return nil, err
	}

	ch, err := loader.Load(chartPath)
	if err != nil {
		return nil, err
	}

	return ch, nil
}

// InstallConfig returns a configured Install action
func (r *Release) InstallConfig(actionConfig *action.Configuration) *action.Install {
	install := action.NewInstall(actionConfig)

	install.RepoURL = r.ChartConfig.RepoURL
	install.ReleaseName = r.ReleaseConfig.Name
	install.Version = r.ChartConfig.Version

	install.Namespace = r.ReleaseConfig.Namespace
	install.CreateNamespace = true

	install.Wait = true
	install.Timeout = 5 * time.Minute

	return install
}

// Install performs a Helm install operation
func (r *Release) Install() (*Release, error) {
	actionConfig, err := r.GetActionConfig()
	if err != nil {
		return nil, err
	}

	install := r.InstallConfig(actionConfig)

	ch, err := r.GetChart(install.ChartPathOptions)
	if err != nil {
		return nil, err
	}

	release, err := install.Run(ch, r.ReleaseConfig.Values)
	if err != nil {
		return nil, err
	}

	r.Revision = release.Version
	return r, nil
}

// UpgradeConfig returns a configured Upgrade action
func (r *Release) UpgradeConfig(actionConfig *action.Configuration) *action.Upgrade {
	upgrade := action.NewUpgrade(actionConfig)

	upgrade.Install = true
	upgrade.RepoURL = r.ChartConfig.RepoURL
	upgrade.Version = r.ChartConfig.Version

	upgrade.Namespace = r.ReleaseConfig.Namespace
	upgrade.ResetValues = true
	upgrade.Wait = true
	upgrade.Timeout = 5 * time.Minute

	return upgrade
}

// Upgrade performs a Helm upgrade operation
func (r *Release) Upgrade() (*Release, error) {
	// Check if there are any changes
	hasDiff, err := r.HasDiff()
	if err != nil {
		return nil, fmt.Errorf("failed to check for differences: %w", err)
	}

	// If no differences, skip upgrade
	if !hasDiff {
		log.Info("No changes detected, skipping upgrade", "name", r.ReleaseConfig.Name)
		// Get current version from deployed release
		deployedRelease, err := r.GetDeployedRelease()
		if err != nil {
			return nil, fmt.Errorf("failed to get deployed release: %w", err)
		}
		r.Revision = deployedRelease.Version
		return r, nil
	}

	actionConfig, err := r.GetActionConfig()
	if err != nil {
		return nil, err
	}

	upgrade := r.UpgradeConfig(actionConfig)

	ch, err := r.GetChart(upgrade.ChartPathOptions)
	if err != nil {
		return nil, err
	}

	release, err := upgrade.Run(r.ReleaseConfig.Name, ch, r.ReleaseConfig.Values)
	if err != nil {
		return nil, err
	}

	r.Revision = release.Version
	return r, nil
}

// Exists checks if a release exists
func (r *Release) Exists() (bool, error) {
	actionConfig, err := r.GetActionConfig()
	if err != nil {
		return false, err
	}

	history := action.NewHistory(actionConfig)
	history.Max = 1

	_, err = history.Run(r.ReleaseConfig.Name)
	return err == nil, nil
}

// Uninstall removes a Helm release
func (r *Release) Uninstall() error {
	actionConfig, err := r.GetActionConfig()
	if err != nil {
		return err
	}

	uninstall := action.NewUninstall(actionConfig)
	_, err = uninstall.Run(r.ReleaseConfig.Name)
	if err != nil {
		return fmt.Errorf("failed to uninstall release %s: %w", r.ReleaseConfig.Name, err)
	}

	log.Info("Successfully uninstalled release", "name", r.ReleaseConfig.Name)
	return nil
}

// Deploy installs or upgrades the release based on whether it exists
func (r *Release) Deploy() error {
	exists, err := r.Exists()
	if err != nil {
		return err
	}

	if exists {
		_, err = r.Upgrade()
		return err
	}

	_, err = r.Install()
	return err
}

// GetDeployedRelease retrieves the currently deployed release
func (r *Release) GetDeployedRelease() (*release.Release, error) {
	actionConfig, err := r.GetActionConfig()
	if err != nil {
		return nil, err
	}

	get := action.NewGet(actionConfig)
	return get.Run(r.ReleaseConfig.Name)
}

// GetDeployedManifests retrieves the manifests of the currently deployed release
func (r *Release) GetDeployedManifests() (string, error) {
	deployedRelease, err := r.GetDeployedRelease()
	if err != nil {
		return "", err
	}

	return deployedRelease.Manifest, nil
}

// GetTemplatedManifests renders the chart templates with the provided values
func (r *Release) GetTemplatedManifests() (string, error) {
	actionConfig, err := r.GetActionConfig()
	if err != nil {
		return "", err
	}

	// Use upgrade with dry-run to get the manifests
	upgrade := r.UpgradeConfig(actionConfig)
	upgrade.DryRun = true

	ch, err := r.GetChart(upgrade.ChartPathOptions)
	if err != nil {
		return "", err
	}

	rel, err := upgrade.Run(r.ReleaseConfig.Name, ch, r.ReleaseConfig.Values)
	if err != nil {
		return "", err
	}

	return rel.Manifest, nil
}

// HasDiff compares deployed manifests with templated manifests
func (r *Release) HasDiff() (bool, error) {
	// Check if release exists
	exists, err := r.Exists()
	if err != nil {
		return false, err
	}

	// If release doesn't exist, there's always a diff (new installation)
	if !exists {
		return true, nil
	}

	// Get deployed manifests
	deployedManifests, err := r.GetDeployedManifests()
	if err != nil {
		return false, fmt.Errorf("failed to get deployed manifests: %w", err)
	}

	// Get templated manifests
	templatedManifests, err := r.GetTemplatedManifests()
	if err != nil {
		return false, fmt.Errorf("failed to get templated manifests: %w", err)
	}

	// Compare manifests
	return !bytes.Equal([]byte(deployedManifests), []byte(templatedManifests)), nil
}

// GetDiff returns the diff between deployed and templated manifests
func (r *Release) GetDiff() (string, error) {
	// Check if release exists
	exists, err := r.Exists()
	if err != nil {
		return "", err
	}

	// If release doesn't exist, return templated manifests as the diff
	if !exists {
		templatedManifests, err := r.GetTemplatedManifests()
		if err != nil {
			return "", fmt.Errorf("failed to get templated manifests: %w", err)
		}
		return fmt.Sprintf("New installation:\n%s", templatedManifests), nil
	}

	// Get deployed manifests
	deployedManifests, err := r.GetDeployedManifests()
	if err != nil {
		return "", fmt.Errorf("failed to get deployed manifests: %w", err)
	}

	// Get templated manifests
	templatedManifests, err := r.GetTemplatedManifests()
	if err != nil {
		return "", fmt.Errorf("failed to get templated manifests: %w", err)
	}

	// For now, return a simple comparison
	if deployedManifests == templatedManifests {
		return "No changes detected", nil
	}

	return fmt.Sprintf("Changes detected between deployed and new manifests"), nil
}
