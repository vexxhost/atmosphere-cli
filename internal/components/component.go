package components

import (
	"github.com/vexxhost/atmosphere/internal/helm"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Component represents a deployable component in the atmosphere system
type Component interface {
	// GetRelease returns the Helm release configuration for this component
	GetRelease(configFlags *genericclioptions.ConfigFlags) *helm.Release
}