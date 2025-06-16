package components

import (
	"context"

	flow "github.com/noneback/go-taskflow"
	"github.com/spf13/viper"
	"github.com/vexxhost/atmosphere/internal/atmosphere"
	"github.com/vexxhost/atmosphere/internal/config"
	"github.com/vexxhost/atmosphere/internal/helm"
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

// GetRelease returns the Helm release configuration for metrics-server
func (m *MetricsServer) GetRelease(ctx context.Context) *helm.Release {
	sectionName := "metrics-server"
	section := atmosphere.ConfigSection(ctx, sectionName)

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

	configFlags := atmosphere.MustConfigFlags(ctx)
	return &helm.Release{
		RESTClientGetter: configFlags,
		ChartConfig:      config.ChartConfigFromConfigSection(section),
		ReleaseConfig:    config.ReleaseConfigFromConfigSection(section),
	}
}

// GetTask returns a task for deploying metrics-server
func (m *MetricsServer) GetTask(ctx context.Context, tf *flow.TaskFlow) *flow.Task {
	return m.HelmComponent.GetTask(ctx, tf, m)
}
