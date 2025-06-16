package components

import (
	"context"

	"github.com/charmbracelet/log"
	flow "github.com/noneback/go-taskflow"
	"github.com/vexxhost/atmosphere/internal/helm"
)

// Component represents a deployable component in the atmosphere system
type Component interface {
	// GetTask returns a task or subflow for deploying this component
	GetTask(ctx context.Context, tf *flow.TaskFlow) *flow.Task
}

// HelmReleaseProvider is the interface that helm components must implement
type HelmReleaseProvider interface {
	GetRelease(ctx context.Context) *helm.Release
}

// HelmComponent provides a default GetTask implementation for Helm-based components
type HelmComponent struct {
	Name string
}

// GetTask returns a standard deployment task for Helm components
func (h *HelmComponent) GetTask(ctx context.Context, tf *flow.TaskFlow, component HelmReleaseProvider) *flow.Task {
	return tf.NewTask("deploy-"+h.Name, func() {
		release := component.GetRelease(ctx)

		log.Info("Deploying component", "name", h.Name)

		if err := release.Deploy(); err != nil {
			log.Fatal("Failed to deploy component", "name", h.Name, "error", err)
		}

		log.Info("Successfully deployed component", "name", h.Name)
	})
}
