package tasks

import (
	flow "github.com/noneback/go-taskflow"
	"github.com/vexxhost/atmosphere/internal/helm"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func NewDeployMetricsServerTask(tf *flow.TaskFlow, configFlags *genericclioptions.ConfigFlags) *flow.Task {
	return DeployHelmChartTask(tf, &helm.Release{
		RESTClientGetter: configFlags,
		RepoURL:          "https://kubernetes-sigs.github.io/metrics-server",
		ChartName:        "metrics-server",
		ChartVersion:     "3.12.2",
		Namespace:        "kube-system",
		Name:             "metrics-server",
		Values: map[string]interface{}{
			"args": []string{
				"--kubelet-insecure-tls",
			},
		},
	})
}
