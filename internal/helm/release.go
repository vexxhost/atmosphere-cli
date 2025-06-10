package helm

import (
	"os"
	"time"

	"github.com/charmbracelet/log"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Release represents a Helm release manager that handles install/upgrade operations
type Release struct {
	Namespace string
	Name      string

	RepoURL      string
	ChartName    string
	ChartVersion string

	RESTClientGetter genericclioptions.RESTClientGetter
	Values           map[string]interface{}
	Version          int
}

func (r *Release) GetActionConfig() (*action.Configuration, error) {
	registryClient, err := registry.NewClient()
	if err != nil {
		return nil, err
	}

	actionConfig := new(action.Configuration)
	actionConfig.RegistryClient = registryClient

	if err := actionConfig.Init(r.RESTClientGetter, r.Namespace, os.Getenv("HELM_DRIVER"), func(format string, args ...interface{}) {
		log.With("namespace", r.Namespace).With("release", r.Name).With("version", r.ChartVersion).Debugf(format, args...)
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
	chartPath, err := chartPathOptions.LocateChart(r.ChartName, cli.New())
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

	install.RepoURL = r.RepoURL
	install.ReleaseName = r.Name
	install.Version = r.ChartVersion

	install.Namespace = r.Namespace
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

	release, err := install.Run(ch, r.Values)
	if err != nil {
		return nil, err
	}

	r.Version = release.Version
	return r, nil
}

// UpgradeConfig returns a configured Upgrade action
func (r *Release) UpgradeConfig(actionConfig *action.Configuration) *action.Upgrade {
	upgrade := action.NewUpgrade(actionConfig)

	upgrade.Install = true
	upgrade.RepoURL = r.RepoURL
	upgrade.Version = r.ChartVersion

	upgrade.Namespace = r.Namespace
	upgrade.ResetValues = true
	upgrade.Wait = true
	upgrade.Timeout = 5 * time.Minute

	return upgrade
}

// Upgrade performs a Helm upgrade operation
func (r *Release) Upgrade() (*Release, error) {
	actionConfig, err := r.GetActionConfig()
	if err != nil {
		return nil, err
	}

	upgrade := r.UpgradeConfig(actionConfig)

	ch, err := r.GetChart(upgrade.ChartPathOptions)
	if err != nil {
		return nil, err
	}

	release, err := upgrade.Run(r.Name, ch, r.Values)
	if err != nil {
		return nil, err
	}

	r.Version = release.Version
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

	_, err = history.Run(r.Name)
	return err == nil, nil
}
