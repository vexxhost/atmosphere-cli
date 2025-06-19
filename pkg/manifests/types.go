package manifests

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
)

// ManifestSource defines the source of a Kubernetes manifest
type ManifestSource struct {
	// Path to a local file containing Kubernetes manifests (supports glob patterns)
	Path string `koanf:"path"`

	// URL to a remote Kubernetes manifest
	URL string `koanf:"url"`

	// Raw manifest content in YAML format
	Content string `koanf:"content"`

	// Path to a Kustomize directory
	KustomizePath string `koanf:"kustomizePath"`

	// Raw kustomization content in YAML format
	KustomizeContent string `koanf:"kustomizeContent"`

	// Template string that can be processed using Go templates
	Template string `koanf:"template"`

	// Values to use when processing the template
	TemplateValues map[string]interface{} `koanf:"templateValues"`
}

// ManifestConfig defines the configuration for a manifest component
type ManifestConfig struct {
	// Namespace to apply the manifests to (will override namespace specified in manifest)
	Namespace string `koanf:"namespace"`

	// Whether to create the namespace if it doesn't exist (defaults to true like Helm)
	CreateNamespace bool `koanf:"createNamespace"`

	// Sources is a list of manifest sources to apply
	Sources []ManifestSource `koanf:"sources"`
}

// Client provides methods for applying Kubernetes manifests
type Client struct {
	dynamicClient dynamic.Interface
	restMapper    meta.RESTMapper
	namespace     string
}
