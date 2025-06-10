package workflows

import (
	flow "github.com/noneback/go-taskflow"
	"github.com/vexxhost/atmosphere/internal/tasks"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// CreateDeployWorkflow creates and returns a deployment TaskFlow
func CreateDeployWorkflow(configFlags *genericclioptions.ConfigFlags) *flow.TaskFlow {
	tf := flow.NewTaskFlow("deploy")

	tasks.NewDeployMetricsServerTask(tf, configFlags)

	return tf
}
