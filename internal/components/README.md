# Components

This package provides a modular component system for deploying services using Helm.

## Adding a New Component

To add a new component, follow these steps:

1. Create a new file in the `internal/components` directory (e.g., `my_component.go`)
2. Implement the `Component` interface:

```go
package components

import (
	"github.com/vexxhost/atmosphere/internal/helm"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// MyComponent represents your custom component
type MyComponent struct {
	*BaseComponent
	chartVersion string
}

// NewMyComponent creates a new MyComponent instance
func NewMyComponent() *MyComponent {
	return &MyComponent{
		BaseComponent: NewBaseComponent("my-component", "my-namespace"),
		chartVersion:  "1.0.0",
	}
}

// GetHelmRelease returns the Helm release configuration
func (m *MyComponent) GetHelmRelease(configFlags *genericclioptions.ConfigFlags) *helm.Release {
	return &helm.Release{
		RESTClientGetter: configFlags,
		RepoURL:          "https://charts.example.com",
		ChartName:        "my-chart",
		ChartVersion:     m.chartVersion,
		Namespace:        m.namespace,
		Name:             m.name,
		Values:           m.getValues(),
	}
}

// getValues returns the Helm values for the component
func (m *MyComponent) getValues() map[string]interface{} {
	return map[string]interface{}{
		// Add your Helm values here
		"key": "value",
	}
}
```

3. Register the component in `registry.go`:

```go
func DefaultRegistry() *Registry {
	registry := NewRegistry()
	
	// Register all available components
	registry.Register(NewCertManager())
	registry.Register(NewMetricsServer())
	registry.Register(NewMemcached())
	registry.Register(NewMyComponent()) // Add your component here
	
	return registry
}
```

## Component Types

Components can deploy from different Helm chart sources:

- **Repository Charts**: Use `RepoURL` + `ChartName` + `ChartVersion`
- **OCI Registry Charts**: Set `ChartName` to the full OCI URL (e.g., `oci://registry-1.docker.io/bitnami/chart:version`)
- **Local Charts**: Set `ChartName` to the local path (e.g., `./charts/my-chart`)

## Component Dependencies

To add dependencies between components, use the `SetDependencies` method:

```go
myComponent := NewMyComponent()
myComponent.SetDependencies([]string{"cert-manager", "metrics-server"})
```