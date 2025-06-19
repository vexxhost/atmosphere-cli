package components

import (
	"context"

	"dario.cat/mergo"
	"github.com/charmbracelet/log"
	flow "github.com/noneback/go-taskflow"
	"github.com/vexxhost/atmosphere/internal/atmosphere"
	"github.com/vexxhost/atmosphere/pkg/manifests"
)

// ManifestComponent implements Component for Kubernetes manifests
type ManifestComponent struct {
	Name       string
	BaseConfig *manifests.ManifestConfig
	Overrides  *manifests.ManifestConfig
}

func NewManifestComponent(name string, baseConfig *manifests.ManifestConfig, overrides *manifests.ManifestConfig) *ManifestComponent {
	return &ManifestComponent{
		Name:       name,
		BaseConfig: baseConfig,
		Overrides:  overrides,
	}
}

// MergedConfig merges the base config with any overrides
func (m *ManifestComponent) MergedConfig() (*manifests.ManifestConfig, error) {
	config := &manifests.ManifestConfig{}
	if err := mergo.Merge(config, m.BaseConfig); err != nil {
		return nil, err
	}

	if m.Overrides != nil {
		if err := mergo.Merge(config, m.Overrides, mergo.WithOverride); err != nil {
			return nil, err
		}
	}

	return config, nil
}

// GetTask implements Component for ManifestComponent
func (m *ManifestComponent) GetTask(ctx context.Context, tf *flow.TaskFlow) *flow.Task {
	componentConfig, err := m.MergedConfig()
	if err != nil {
		log.Fatal("Failed to merge component configuration", "name", m.Name, "error",
			err)
		return nil
	}

	return tf.NewTask("deploy-"+m.Name, func() {
		log.Info("Deploying manifests", "name", m.Name, "namespace", componentConfig.Namespace)

		// Get REST client getter from context
		getter := atmosphere.MustConfigFlags(ctx)

		// First create the namespace if needed (do this separately to ensure namespace exists)
		if componentConfig.Namespace != "" {
			// Create a client just for namespace operations
			nsClient, err := manifests.NewClient(getter, "")
			if err != nil {
				log.Fatal("Failed to create namespace client", "name", m.Name, "error", err)
			}

			// Pre-create the namespace to ensure it exists
			if err := nsClient.EnsureNamespaceExists(ctx, componentConfig.Namespace); err != nil {
				log.Fatal("Failed to create namespace", "namespace", componentConfig.Namespace, "error", err)
			}
		}

		// Now apply the manifests within the namespace
		client, err := manifests.NewClient(getter, componentConfig.Namespace)
		if err != nil {
			log.Fatal("Failed to create manifest client", "name", m.Name, "error", err)
		}

		if err := client.ApplyManifests(ctx, componentConfig); err != nil {
			log.Fatal("Failed to apply manifests", "name", m.Name, "error", err)
		}

		log.Info("Successfully deployed manifests", "name", m.Name)
	})
}
