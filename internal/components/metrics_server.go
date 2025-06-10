package components

import (
	"github.com/vexxhost/atmosphere/internal/helm"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// MetricsServer represents the metrics-server component
type MetricsServer struct {
	*BaseComponent
	chartVersion string
}

// NewMetricsServer creates a new MetricsServer component
func NewMetricsServer() *MetricsServer {
	return &MetricsServer{
		BaseComponent: NewBaseComponent("metrics-server", "kube-system"),
		chartVersion:  "3.12.2",
	}
}

// GetHelmRelease returns the Helm release configuration
func (m *MetricsServer) GetHelmRelease(configFlags *genericclioptions.ConfigFlags) *helm.Release {
	return &helm.Release{
		RESTClientGetter: configFlags,
		RepoURL:          "https://kubernetes-sigs.github.io/metrics-server",
		ChartName:        "metrics-server",
		ChartVersion:     m.chartVersion,
		Namespace:        m.namespace,
		Name:             m.name,
		Values:           m.getValues(),
	}
}

// getValues returns the Helm values for metrics-server
func (m *MetricsServer) getValues() map[string]interface{} {
	return map[string]interface{}{
		"args": []string{
			"--kubelet-insecure-tls",
		},
	}
}