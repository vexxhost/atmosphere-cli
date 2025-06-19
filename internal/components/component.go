package components

import (
	"context"

	"dario.cat/mergo"
	"github.com/charmbracelet/log"
	flow "github.com/noneback/go-taskflow"
	"github.com/vexxhost/atmosphere/internal/atmosphere"
	"github.com/vexxhost/atmosphere/pkg/helm"
)

type Component interface {
	GetTask(ctx context.Context, tf *flow.TaskFlow) *flow.Task
}

type HelmComponent struct {
	Name       string
	BaseConfig *helm.ComponentConfig
	Overrides  *helm.ComponentConfig
}

func (h *HelmComponent) MergedConfig() (*helm.ComponentConfig, error) {
	config := &helm.ComponentConfig{}
	if err := mergo.Merge(config, h.BaseConfig); err != nil {
		return nil, err
	}

	if h.Overrides != nil {
		if err := mergo.Merge(config, h.Overrides, mergo.WithOverride); err != nil {
			return nil, err
		}
	}

	return config, nil
}

func (h *HelmComponent) GetTask(ctx context.Context, tf *flow.TaskFlow) *flow.Task {
	componentConfig, err := h.MergedConfig()
	if err != nil {
		log.Fatal("Failed to merge component configuration", "name", h.Name, "error",
			err)
		return nil
	}

	return tf.NewTask("deploy-"+h.Name, func() {
		log.Info("Deploying component", "name", h.Name)

		// Get REST client getter from context
		getter := atmosphere.MustConfigFlags(ctx)
		client, err := helm.NewClient(getter, componentConfig.Release.Namespace)
		if err != nil {
			log.Fatal("Failed to create helm client", "name", h.Name, "error", err)
		}

		if _, err := client.DeployRelease(componentConfig); err != nil {
			log.Fatal("Failed to deploy component", "name", h.Name, "error", err)
		}

		log.Info("Successfully deployed component", "name", h.Name)
	})
}
