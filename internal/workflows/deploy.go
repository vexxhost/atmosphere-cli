package workflows

import (
	flow "github.com/noneback/go-taskflow"
	"github.com/vexxhost/atmosphere/internal/components"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// CreateDeployWorkflow creates and returns a deployment TaskFlow
func CreateDeployWorkflow(configFlags *genericclioptions.ConfigFlags) *flow.TaskFlow {
	// Create a new custom flow
	tf := NewTaskFlow()

	// Get the component registry
	registry := components.DefaultRegistry()

	// Deploy all enabled components
	for _, component := range registry.GetEnabled() {
		helmRelease := component.GetHelmRelease(configFlags)
		tf.NewDeployHelmChartFlow(helmRelease)
	}

	// Return the underlying TaskFlow for execution
	return tf.TaskFlow
}

// CreateDeployWorkflowWithComponents creates a deployment workflow with specific components
func CreateDeployWorkflowWithComponents(configFlags *genericclioptions.ConfigFlags, componentNames []string) (*flow.TaskFlow, error) {
	// Create a new custom flow
	tf := NewTaskFlow()

	// Get the component registry
	registry := components.DefaultRegistry()

	// Deploy only specified components
	for _, name := range componentNames {
		component, err := registry.Get(name)
		if err != nil {
			return nil, err
		}

		if component.IsEnabled() {
			helmRelease := component.GetHelmRelease(configFlags)
			tf.NewDeployHelmChartFlow(helmRelease)
		}
	}

	// Return the underlying TaskFlow for execution
	return tf.TaskFlow, nil
}
