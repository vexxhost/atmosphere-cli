package config

import (
	"github.com/spf13/viper"
)

type ChartConfig struct {
	// RepoURL points to the Helm chart repository URL.
	RepoURL string

	// Name is the name of the Helm chart.
	Name string

	// Version is the version of the Helm chart.
	Version string
}

func ChartConfigFromConfigSection(section *viper.Viper) *ChartConfig {
	if section == nil {
		return &ChartConfig{}
	}

	return &ChartConfig{
		RepoURL: section.GetString("chart.repository"),
		Name:    section.GetString("chart.name"),
		Version: section.GetString("chart.version"),
	}
}

type ReleaseConfig struct {
	// Namespace is the Kubernetes namespace where the Helm release will be deployed.
	Namespace string

	// Name is the name of the Helm release.
	Name string

	// Values are the Helm values to be used for the release.
	Values map[string]interface{}
}

func ReleaseConfigFromConfigSection(section *viper.Viper) *ReleaseConfig {
	if section == nil {
		return &ReleaseConfig{}
	}

	allSettings := section.AllSettings()
	var values map[string]interface{}

	if release, ok := allSettings["release"].(map[string]interface{}); ok {
		if releaseValues, ok := release["values"].(map[string]interface{}); ok {
			values = releaseValues
		}
	}

	return &ReleaseConfig{
		Namespace: section.GetString("release.namespace"),
		Name:      section.GetString("release.name"),
		Values:    values,
	}
}
