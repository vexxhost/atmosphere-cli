package components

import (
	"context"

	flow "github.com/noneback/go-taskflow"
	"github.com/spf13/viper"
	"github.com/vexxhost/atmosphere/internal/atmosphere"
	"github.com/vexxhost/atmosphere/pkg/helm"
)

// MetricsServer represents the metrics-server component
type MetricsServer struct {
	*HelmComponent
}

// NewMetricsServer creates a new MetricsServer component
func NewMetricsServer() *MetricsServer {
	return &MetricsServer{
		HelmComponent: &HelmComponent{
			Name: "metrics-server",
		},
	}
}

// GetChartConfig returns the Helm chart configuration for metrics-server
func (m *MetricsServer) GetChartConfig(ctx context.Context) *helm.ChartConfig {
	sectionName := "metrics-server"
	section := atmosphere.ConfigSection(ctx, sectionName)

	// Set defaults if no config section exists
	if section == nil {
		viper.Set(sectionName, map[string]interface{}{})
		section = viper.Sub(sectionName)
	}

	// Set defaults directly on the section
	section.SetDefault("chart.repository", "https://kubernetes-sigs.github.io/metrics-server")
	section.SetDefault("chart.name", "metrics-server")
	section.SetDefault("chart.version", "3.12.2")

	// Create helm.ChartConfig from section
	return &helm.ChartConfig{
		RepoURL: section.GetString("chart.repository"),
		Name:    section.GetString("chart.name"),
		Version: section.GetString("chart.version"),
	}
}

// GetReleaseConfig returns the Helm release configuration for metrics-server
func (m *MetricsServer) GetReleaseConfig(ctx context.Context) *helm.ReleaseConfig {
	sectionName := "metrics-server"
	section := atmosphere.ConfigSection(ctx, sectionName)

	// Set defaults if no config section exists
	if section == nil {
		viper.Set(sectionName, map[string]interface{}{})
		section = viper.Sub(sectionName)
	}

	// Set defaults directly on the section
	section.SetDefault("release.namespace", "kube-system")
	section.SetDefault("release.name", "metrics-server")
	section.SetDefault("release.values", map[string]interface{}{
		"args": []string{
			"--kubelet-insecure-tls",
		},
	})

	// Create helm.ReleaseConfig from section
	allSettings := section.AllSettings()
	var values map[string]interface{}
	
	if release, ok := allSettings["release"].(map[string]interface{}); ok {
		if releaseValues, ok := release["values"].(map[string]interface{}); ok {
			values = releaseValues
		}
	}
	
	return &helm.ReleaseConfig{
		Namespace: section.GetString("release.namespace"),
		Name:      section.GetString("release.name"),
		Values:    values,
	}
}

// GetTask returns a task for deploying metrics-server
func (m *MetricsServer) GetTask(ctx context.Context, tf *flow.TaskFlow) *flow.Task {
	return m.HelmComponent.GetTask(ctx, tf, m)
}