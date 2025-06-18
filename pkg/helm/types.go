package helm

// ChartConfig contains the configuration for a Helm chart
type ChartConfig struct {
	// RepoURL points to the Helm chart repository URL.
	RepoURL string `koanf:"repository"`

	// Name is the name of the Helm chart.
	Name string `koanf:"name"`

	// Version is the version of the Helm chart.
	Version string `koanf:"version"`
}

// ReleaseConfig contains the configuration for a Helm release
type ReleaseConfig struct {
	// Namespace is the Kubernetes namespace where the Helm release will be deployed.
	Namespace string `koanf:"namespace"`

	// Name is the name of the Helm release.
	Name string `koanf:"name"`

	// Values are the Helm values to be used for the release.
	Values map[string]interface{} `koanf:"values"`
}

// ComponentConfig contains both chart and release configuration for a component
type ComponentConfig struct {
	// Chart contains the chart configuration
	Chart *ChartConfig `koanf:"chart"`

	// Release contains the release configuration
	Release *ReleaseConfig `koanf:"release"`
}
