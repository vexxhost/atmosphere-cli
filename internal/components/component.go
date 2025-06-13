package components

import (
	"github.com/charmbracelet/log"
	flow "github.com/noneback/go-taskflow"
	"github.com/vexxhost/atmosphere/internal/helm"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Component represents a deployable component in the atmosphere system
type Component interface {
	// GetTask returns a task or subflow for deploying this component
	GetTask(tf *flow.TaskFlow, configFlags *genericclioptions.ConfigFlags) *flow.Task
}

// HelmReleaseProvider is the interface that helm components must implement
type HelmReleaseProvider interface {
	GetRelease(configFlags *genericclioptions.ConfigFlags) *helm.Release
}

// HelmComponent provides a default GetTask implementation for Helm-based components
type HelmComponent struct {
	Name string
}

// GetTask returns a standard deployment task for Helm components
func (h *HelmComponent) GetTask(tf *flow.TaskFlow, configFlags *genericclioptions.ConfigFlags, component HelmReleaseProvider) *flow.Task {
	return tf.NewTask("deploy-"+h.Name, func() {
		release := component.GetRelease(configFlags)
		
		log.Info("Deploying component", "name", h.Name)
		
		if err := release.Deploy(); err != nil {
			log.Fatal("Failed to deploy component", "name", h.Name, "error", err)
		}
		
		log.Info("Successfully deployed component", "name", h.Name)
	})
}