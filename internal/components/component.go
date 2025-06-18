package components

import (
	"context"

	"github.com/charmbracelet/log"
	flow "github.com/noneback/go-taskflow"
	"github.com/vexxhost/atmosphere/internal/atmosphere"
	"github.com/vexxhost/atmosphere/pkg/helm"
)

// Component represents a deployable component in the atmosphere system
type Component interface {
	// GetTask returns a task or subflow for deploying this component
	GetTask(ctx context.Context, tf *flow.TaskFlow) *flow.Task
}

// HelmReleaseProvider is the interface that helm components must implement
type HelmReleaseProvider interface {
	GetChartConfig(ctx context.Context) *helm.ChartConfig
	GetReleaseConfig(ctx context.Context) *helm.ReleaseConfig
}

// HelmComponent provides a default GetTask implementation for Helm-based components
type HelmComponent struct {
	Name string
}

// GetTask returns a standard deployment task for Helm components
func (h *HelmComponent) GetTask(ctx context.Context, tf *flow.TaskFlow, component HelmReleaseProvider) *flow.Task {
	return tf.NewTask("deploy-"+h.Name, func() {
		chartConfig := component.GetChartConfig(ctx)
		releaseConfig := component.GetReleaseConfig(ctx)

		log.Info("Deploying component", "name", h.Name)

		// Get REST client getter from context
		configFlags := atmosphere.MustConfigFlags(ctx)
		client, err := helm.NewClient(configFlags, releaseConfig.Namespace)
		if err != nil {
			log.Fatal("Failed to create helm client", "name", h.Name, "error", err)
		}

		if _, err := client.DeployRelease(chartConfig, releaseConfig); err != nil {
			log.Fatal("Failed to deploy component", "name", h.Name, "error", err)
		}

		log.Info("Successfully deployed component", "name", h.Name)
	})
}
