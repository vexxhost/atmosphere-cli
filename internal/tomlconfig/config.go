package tomlconfig

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/log"
)

// CaseSensitiveConfig holds the full parsed TOML config with preserved case
var CaseSensitiveConfig map[string]interface{}

// AtmosphereConfig represents the top-level configuration structure
type AtmosphereConfig struct {
	MetricsServer map[string]interface{} `toml:"metrics-server"`
	Vault         map[string]interface{} `toml:"vault"`
}

// LoadConfig loads configuration from TOML files
func LoadConfig() (*AtmosphereConfig, error) {
	config := &AtmosphereConfig{}

	paths := []string{
		"atmosphere.toml",
		"/etc/atmosphere/atmosphere.toml",
	}

	var configFile string
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			configFile = path
			break
		}
	}

	if configFile == "" {
		log.Debug("No config file found, using defaults")
		return config, nil
	}

	log.Debug("Using config file", "file", configFile)

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	// Parse TOML into generic map to preserve case
	if _, err := toml.Decode(string(data), &CaseSensitiveConfig); err != nil {
		return nil, err
	}

	// Also parse into typed struct for convenience
	if _, err := toml.Decode(string(data), config); err != nil {
		return nil, err
	}

	return config, nil
}

// ChartConfig represents the Helm chart configuration
type ChartConfig struct {
	// RepoURL points to the Helm chart repository URL.
	RepoURL string

	// Name is the name of the Helm chart.
	Name string

	// Version is the version of the Helm chart.
	Version string
}

// ReleaseConfig represents the Helm release configuration
type ReleaseConfig struct {
	// Namespace is the Kubernetes namespace where the Helm release will be deployed.
	Namespace string

	// Name is the name of the Helm release.
	Name string

	// Values are the Helm values to be used for the release.
	Values map[string]interface{}
}

// GetChartConfig extracts chart configuration from a component section
func GetChartConfig(componentConfig map[string]interface{}) *ChartConfig {
	config := &ChartConfig{}

	if chartSection, ok := componentConfig["chart"].(map[string]interface{}); ok {
		if repository, ok := chartSection["repository"].(string); ok {
			config.RepoURL = repository
		}
		if name, ok := chartSection["name"].(string); ok {
			config.Name = name
		}
		if version, ok := chartSection["version"].(string); ok {
			config.Version = version
		}
	}

	return config
}

// GetReleaseConfig extracts release configuration from a component section
func GetReleaseConfig(componentConfig map[string]interface{}) *ReleaseConfig {
	config := &ReleaseConfig{
		Values: make(map[string]interface{}),
	}

	if releaseSection, ok := componentConfig["release"].(map[string]interface{}); ok {
		if namespace, ok := releaseSection["namespace"].(string); ok {
			config.Namespace = namespace
		}
		if name, ok := releaseSection["name"].(string); ok {
			config.Name = name
		}
		if values, ok := releaseSection["values"].(map[string]interface{}); ok {
			config.Values = values
		}
	}

	return config
}

// GetComponentConfig returns the component config with original case preserved
func GetComponentConfig(componentName string) map[string]interface{} {
	if CaseSensitiveConfig == nil {
		return nil
	}

	componentConfig, ok := CaseSensitiveConfig[componentName]
	if !ok {
		return nil
	}

	configMap, ok := componentConfig.(map[string]interface{})
	if !ok {
		return nil
	}

	return configMap
}

// GetReleaseValues returns the release values with original case preserved
func GetReleaseValues(componentName string) map[string]interface{} {
	componentConfig := GetComponentConfig(componentName)
	if componentConfig == nil {
		return nil
	}

	release, ok := componentConfig["release"]
	if !ok {
		return nil
	}

	releaseMap, ok := release.(map[string]interface{})
	if !ok {
		return nil
	}

	values, ok := releaseMap["values"]
	if !ok {
		return nil
	}

	valuesMap, ok := values.(map[string]interface{})
	if !ok {
		return nil
	}

	return valuesMap
}
