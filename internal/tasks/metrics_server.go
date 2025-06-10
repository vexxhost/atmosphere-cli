package tasks

import (
	flow "github.com/noneback/go-taskflow"
	"github.com/spf13/viper"
	"github.com/vexxhost/atmosphere/internal/config"
	"github.com/vexxhost/atmosphere/internal/helm"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func NewDeployMetricsServerTask(tf *flow.TaskFlow, configFlags *genericclioptions.ConfigFlags) *flow.Task {
	sectionName := "metrics-server"
	section := viper.Sub(sectionName)

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

	return DeployHelmChartTask(tf, &helm.Release{
		RESTClientGetter: configFlags,
		ChartConfig:      config.ChartConfigFromConfigSection(section),
		ReleaseConfig:    config.ReleaseConfigFromConfigSection(section),
	})
}
