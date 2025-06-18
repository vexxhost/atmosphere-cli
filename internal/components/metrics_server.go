package components

import (
	"github.com/vexxhost/atmosphere/pkg/helm"
)

func NewMetricsServer(overrides *helm.ComponentConfig) *HelmComponent {
	return &HelmComponent{
		Name: "metrics-server",
		BaseConfig: &helm.ComponentConfig{
			Chart: &helm.ChartConfig{
				RepoURL: "https://kubernetes-sigs.github.io/metrics-server",
				Name:    "metrics-server",
				Version: "3.12.2",
			},
			Release: &helm.ReleaseConfig{
				Namespace: "kube-system",
				Name:      "metrics-server",
				Values: map[string]interface{}{
					"args": []string{
						"--kubelet-insecure-tls",
					},
				},
			},
		},
		Overrides: overrides,
	}
}
