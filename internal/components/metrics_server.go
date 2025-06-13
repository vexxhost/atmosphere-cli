package components

import (
	"github.com/spf13/viper"
	"github.com/vexxhost/atmosphere/internal/config"
	"github.com/vexxhost/atmosphere/internal/helm"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// MetricsServer represents the metrics-server component
type MetricsServer struct{}

// NewMetricsServer creates a new MetricsServer component
func NewMetricsServer() *MetricsServer {
	return &MetricsServer{}
}

// GetRelease returns the Helm release configuration for metrics-server
func (m *MetricsServer) GetRelease(configFlags *genericclioptions.ConfigFlags) *helm.Release {
	sectionName := "metrics-server"
	section := viper.Sub(sectionName)

	// Set defaults if no config section exists
	if section == nil {
		viper.Set(sectionName, map[string]interface{}{})
		section = viper.Sub(sectionName)
	}

	helm.SetReleaseDefault(section, &helm.Release{
		ChartConfig: &config.ChartConfig{
			RepoURL: "https://kubernetes-sigs.github.io/metrics-server",
			Name:    "metrics-server",
			Version: "3.12.2",
		},
		ReleaseConfig: &config.ReleaseConfig{
			Namespace: "kube-system",
			Name:      "metrics-server",
			Values: map[string]interface{}{
				"args": []string{
					"--kubelet-insecure-tls",
				},
			},
		},
	})

	return &helm.Release{
		RESTClientGetter: configFlags,
		ChartConfig:      config.ChartConfigFromConfigSection(section),
		ReleaseConfig:    config.ReleaseConfigFromConfigSection(section),
	}
}
