package components

import (
	"github.com/vexxhost/atmosphere/internal/helm"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Component defines the interface for all deployable components
type Component interface {
	// Name returns the component name
	Name() string

	// Namespace returns the target namespace for the component
	Namespace() string

	// GetHelmRelease returns the Helm release configuration for the component
	GetHelmRelease(configFlags *genericclioptions.ConfigFlags) *helm.Release

	// IsEnabled returns whether the component should be deployed
	IsEnabled() bool

	// Dependencies returns a list of component names this component depends on
	Dependencies() []string
}

// BaseComponent provides common functionality for components
type BaseComponent struct {
	name         string
	namespace    string
	enabled      bool
	dependencies []string
}

// NewBaseComponent creates a new base component
func NewBaseComponent(name, namespace string) *BaseComponent {
	return &BaseComponent{
		name:         name,
		namespace:    namespace,
		enabled:      true,
		dependencies: []string{},
	}
}

// Name returns the component name
func (b *BaseComponent) Name() string {
	return b.name
}

// Namespace returns the component namespace
func (b *BaseComponent) Namespace() string {
	return b.namespace
}

// IsEnabled returns whether the component is enabled
func (b *BaseComponent) IsEnabled() bool {
	return b.enabled
}

// SetEnabled sets the enabled state
func (b *BaseComponent) SetEnabled(enabled bool) {
	b.enabled = enabled
}

// Dependencies returns the component dependencies
func (b *BaseComponent) Dependencies() []string {
	return b.dependencies
}

// SetDependencies sets the component dependencies
func (b *BaseComponent) SetDependencies(deps []string) {
	b.dependencies = deps
}