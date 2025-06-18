package helm

// ChartConfig contains the configuration for a Helm chart
type ChartConfig struct {
	// RepoURL points to the Helm chart repository URL.
	RepoURL string

	// Name is the name of the Helm chart.
	Name string

	// Version is the version of the Helm chart.
	Version string
}

// ReleaseConfig contains the configuration for a Helm release
type ReleaseConfig struct {
	// Namespace is the Kubernetes namespace where the Helm release will be deployed.
	Namespace string

	// Name is the name of the Helm release.
	Name string

	// Values are the Helm values to be used for the release.
	Values map[string]interface{}
}
