package components

import (
	"github.com/charmbracelet/log"
	"github.com/vexxhost/atmosphere/internal/config"
	"github.com/vexxhost/atmosphere/internal/helm"
	"github.com/vexxhost/atmosphere/internal/tomlconfig"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Vault represents the vault-operator component using bank-vaults
type Vault struct{}

// NewVault creates a new Vault component
func NewVault() *Vault {
	return &Vault{}
}

// GetRelease returns the Helm release configuration for vault-operator
func (v *Vault) GetRelease(configFlags *genericclioptions.ConfigFlags) *helm.Release {
	sectionName := "vault"

	// Get the vault config directly from our case-preserving TOML parser
	vaultConfig := tomlconfig.GetComponentConfig(sectionName)

	// Default values to use if not specified in config
	defaultValues := map[string]interface{}{
		"image": map[string]interface{}{
			"tag": "v1.22.6",
		},
		"bankVaults": map[string]interface{}{
			"image": map[string]interface{}{
				"tag": "v1.31.4",
			},
		},
	}

	// Default chart config
	chartRegistry := "oci://ghcr.io/bank-vaults/helm-charts"
	chartName := "vault-operator"
	chartVersion := "1.22.6"

	// Default release config
	namespace := "vault"
	releaseName := "vault-operator"

	// Extract config values from case-preserved config
	if vaultConfig != nil {
		log.Debug("Found vault config with preserved case", "config", vaultConfig)

		// Extract chart info
		if chart, ok := vaultConfig["chart"].(map[string]interface{}); ok {
			if registry, ok := chart["registry"].(string); ok && registry != "" {
				chartRegistry = registry
			}
			if name, ok := chart["name"].(string); ok && name != "" {
				chartName = name
			}
			if version, ok := chart["version"].(string); ok && version != "" {
				chartVersion = version
			}
		}

		// Extract release info
		if release, ok := vaultConfig["release"].(map[string]interface{}); ok {
			if ns, ok := release["namespace"].(string); ok && ns != "" {
				namespace = ns
			}
			if name, ok := release["name"].(string); ok && name != "" {
				releaseName = name
			}
		}
	}

	// Get values directly from our case-preserving TOML parser
	values := tomlconfig.GetReleaseValues(sectionName)

	// If no values found or missing keys, apply defaults
	if values == nil {
		values = defaultValues
		log.Debug("No values found in config, using defaults", "values", values)
	} else {
		log.Debug("Found values with preserved case", "values", values)

		// Apply any missing default values that aren't in the config
		for k, v := range defaultValues {
			if _, exists := values[k]; !exists {
				values[k] = v
			}
		}
	}

	// Create chart config with full OCI path
	fullChartName := chartRegistry + "/" + chartName
	chartConfig := &config.ChartConfig{
		Name:    fullChartName,
		Version: chartVersion,
	}

	// Add config hash to enable hash-based diffing
	values = helm.AddConfigHash(values)

	log.Debug("Component values for deployment",
		"component", "vault",
		"replicaCount", values["replicaCount"])

	releaseConfig := &config.ReleaseConfig{
		Namespace: namespace,
		Name:      releaseName,
		Values:    values,
	}

	log.Debug("Using chart config", "config", chartConfig)
	log.Debug("Using release config", "config", releaseConfig, "values", values)

	return &helm.Release{
		RESTClientGetter: configFlags,
		ChartConfig:      chartConfig,
		ReleaseConfig:    releaseConfig,
	}
}
