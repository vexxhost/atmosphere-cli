package components

import (
	"fmt"
)

// Registry manages all available components
type Registry struct {
	components map[string]Component
}

// NewRegistry creates a new component registry
func NewRegistry() *Registry {
	return &Registry{
		components: make(map[string]Component),
	}
}

// Register adds a component to the registry
func (r *Registry) Register(component Component) error {
	name := component.Name()
	if _, exists := r.components[name]; exists {
		return fmt.Errorf("component %s already registered", name)
	}
	r.components[name] = component
	return nil
}

// Get retrieves a component by name
func (r *Registry) Get(name string) (Component, error) {
	component, exists := r.components[name]
	if !exists {
		return nil, fmt.Errorf("component %s not found", name)
	}
	return component, nil
}

// GetAll returns all registered components
func (r *Registry) GetAll() []Component {
	components := make([]Component, 0, len(r.components))
	for _, component := range r.components {
		components = append(components, component)
	}
	return components
}

// GetEnabled returns all enabled components
func (r *Registry) GetEnabled() []Component {
	var enabled []Component
	for _, component := range r.components {
		if component.IsEnabled() {
			enabled = append(enabled, component)
		}
	}
	return enabled
}

// DefaultRegistry creates a registry with all default components
func DefaultRegistry() *Registry {
	registry := NewRegistry()
	
	// Register all available components
	registry.Register(NewCertManager())
	registry.Register(NewMetricsServer())
	registry.Register(NewMemcached())
	
	return registry
}