package helm

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
	"github.com/vexxhost/atmosphere/internal/config"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
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

// IsOCI returns true if the chart is from an OCI registry
func (r *Release) IsOCI() bool {
	return strings.HasPrefix(r.ChartConfig.Name, "oci://")
}

func (r *Release) GetActionConfig() (*action.Configuration, error) {
	// Set up proper namespace handling for OCI charts
	configFlags := genericclioptions.NewConfigFlags(true)
	if r.ReleaseConfig.Namespace != "" {
		configFlags.Namespace = &r.ReleaseConfig.Namespace
	}

	registryClient, err := registry.NewClient()
	if err != nil {
		return nil, err
	}

	actionConfig := new(action.Configuration)
	actionConfig.RegistryClient = registryClient

	if err := actionConfig.Init(configFlags, r.ReleaseConfig.Namespace, os.Getenv("HELM_DRIVER"), func(format string, args ...interface{}) {
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
	settings := cli.New()
	settings.SetNamespace(r.ReleaseConfig.Namespace)

	var chartPath string
	var err error

	if r.IsOCI() {
		log.Debug("Processing OCI chart", "chart", r.ChartConfig.Name, "namespace", r.ReleaseConfig.Namespace)

		// For OCI charts, use the downloader to handle specific OCI requirements
		puller := &downloader.ChartDownloader{
			Out:              os.Stdout,
			Verify:           downloader.VerifyNever,
			Getters:          getter.All(settings),
			RepositoryConfig: settings.RepositoryConfig,
			RepositoryCache:  settings.RepositoryCache,
		}

		// Download the OCI chart to the Helm cache
		chartPath, _, err = puller.DownloadTo(r.ChartConfig.Name, r.ChartConfig.Version, settings.RepositoryCache)
		if err != nil {
			return nil, fmt.Errorf("failed to download OCI chart: %w", err)
		}
	} else {
		// Standard chart repository handling
		chartPath, err = chartPathOptions.LocateChart(r.ChartConfig.Name, settings)
		if err != nil {
			return nil, err
		}
	}

	ch, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	return ch, nil
}

// InstallConfig returns a configured Install action
func (r *Release) InstallConfig(actionConfig *action.Configuration) *action.Install {
	install := action.NewInstall(actionConfig)

	install.RepoURL = r.ChartConfig.RepoURL
	install.ReleaseName = r.ReleaseConfig.Name
	install.Version = r.ChartConfig.Version

	// Ensure namespace is always set explicitly
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

	// Double-check namespace is set
	if install.Namespace == "" {
		install.Namespace = r.ReleaseConfig.Namespace
	}

	var ch *chart.Chart
	chartSource := "HTTP(S)"

	if r.IsOCI() {
		chartSource = "OCI"
		ch, err = r.GetChart(install.ChartPathOptions)
		if err != nil {
			return nil, err
		}
	} else {
		ch, err = r.GetChart(install.ChartPathOptions)
		if err != nil {
			return nil, err
		}
	}

	// Debug log showing what's being deployed
	log.Debug("Helm install", "release", r.ReleaseConfig.Name, "namespace", r.ReleaseConfig.Namespace, "chartSource", chartSource, "values", r.ReleaseConfig.Values)

	// No need for case correction with BurntSushi/toml parser
	values := r.ReleaseConfig.Values

	release, err := install.Run(ch, values)
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

	// Ensure namespace is always set explicitly
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

	// Double-check namespace is set
	if upgrade.Namespace == "" {
		upgrade.Namespace = r.ReleaseConfig.Namespace
	}

	var ch *chart.Chart
	chartSource := "HTTP(S)"

	if r.IsOCI() {
		chartSource = "OCI"
		ch, err = r.GetChart(upgrade.ChartPathOptions)
		if err != nil {
			return nil, err
		}
	} else {
		ch, err = r.GetChart(upgrade.ChartPathOptions)
		if err != nil {
			return nil, err
		}
	}

	// Debug log showing what's being deployed
	log.Debug("Helm upgrade", "release", r.ReleaseConfig.Name, "namespace", r.ReleaseConfig.Namespace, "chartSource", chartSource, "values", r.ReleaseConfig.Values)

	// No need for case correction with BurntSushi/toml parser
	values := r.ReleaseConfig.Values

	release, err := upgrade.Run(r.ReleaseConfig.Name, ch, values)
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

	upgrade := action.NewUpgrade(actionConfig)
	upgrade.DryRun = true
	upgrade.Namespace = r.ReleaseConfig.Namespace

	var ch *chart.Chart
	if r.IsOCI() {
		ch, err = r.GetChart(upgrade.ChartPathOptions)
		if err != nil {
			return "", fmt.Errorf("failed to get OCI chart for templating: %w", err)
		}
	} else {
		ch, err = r.GetChart(upgrade.ChartPathOptions)
		if err != nil {
			return "", fmt.Errorf("failed to get chart for templating: %w", err)
		}
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

	// If release doesn't exist, we need to install it
	if !exists {
		log.Debug("Release not found, will install", "name", r.ReleaseConfig.Name)
		return true, nil
	}

	// Get deployed release
	deployedRelease, err := r.GetDeployedRelease()
	if err != nil {
		log.Debug("Failed to get deployed release, assuming changes needed", "name", r.ReleaseConfig.Name, "error", err)
		return true, nil
	}
	// Use hash-based comparison function
	if NeedsUpdate(r.ReleaseConfig.Values, deployedRelease) {
		return true, nil
	}

	// No changes detected based on hash comparison
	log.Debug("Config hash indicates no changes needed", "name", r.ReleaseConfig.Name)
	return false, nil
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

	return "Changes detected between deployed and new manifests", nil
}
