package components

import (
	"github.com/vexxhost/atmosphere/pkg/manifests"
)

// NewPrometheusStack creates a new component that deploys Prometheus monitoring stack
// using raw manifest files
func NewPrometheusStack(overrides *manifests.ManifestConfig) *ManifestComponent {
	return NewManifestComponent(
		"prometheus-stack",
		&manifests.ManifestConfig{
			Namespace:       "monitoring",
			CreateNamespace: true,
			Sources: []manifests.ManifestSource{
				{
					// Example of using a URL to fetch manifests
					URL: "https://raw.githubusercontent.com/prometheus-community/helm-charts/main/charts/kube-prometheus-stack/crds/crd-prometheuses.yaml",
				},
				{
					// Example of using local file path (supports glob patterns)
					Path: "/path/to/prometheus/*.yaml",
				},
				{
					// Example of inline manifest content
					Content: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus-server
  labels:
    app: prometheus
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus
  template:
    metadata:
      labels:
        app: prometheus
    spec:
      containers:
      - name: prometheus
        image: prom/prometheus:v2.45.0
        ports:
        - containerPort: 9090
---
apiVersion: v1
kind: Service
metadata:
  name: prometheus-service
spec:
  selector:
    app: prometheus
  ports:
  - port: 9090
    targetPort: 9090
`,
				},
			},
		},
		overrides,
	)
}

// NewIngressNginx creates a component for deploying ingress-nginx
// using Kustomize
func NewIngressNginx(overrides *manifests.ManifestConfig) *ManifestComponent {
	return NewManifestComponent(
		"ingress-nginx",
		&manifests.ManifestConfig{
			Namespace:       "ingress-nginx",
			CreateNamespace: true,
			Sources: []manifests.ManifestSource{
				{
					// Example of using Kustomize
					KustomizePath: "/path/to/ingress-nginx/kustomize",
				},
			},
		},
		overrides,
	)
}

// NewConfigMap creates a ConfigMap from a Go template
func NewConfigMap(overrides *manifests.ManifestConfig) *ManifestComponent {
	return NewManifestComponent(
		"config-map",
		&manifests.ManifestConfig{
			Namespace: "manifest-test", // Changed to test namespace creation
			// CreateNamespace is true by default now, so this namespace will be created
			Sources: []manifests.ManifestSource{
				{
					// Example of using Go template
					Template: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Name }}
data:
  {{- range $key, $value := .Data }}
  {{ $key }}: "{{ $value }}"
  {{- end }}
`,
					TemplateValues: map[string]interface{}{
						"Name": "my-config",
						"Data": map[string]string{
							"key1": "value1",
							"key2": "value2",
						},
					},
				},
			},
		},
		overrides,
	)
}

// NewManifestWithKustomize creates a component using inline Kustomization content
func NewManifestWithKustomize(overrides *manifests.ManifestConfig) *ManifestComponent {
	return NewManifestComponent(
		"kustomize-inline",
		&manifests.ManifestConfig{
			Namespace:       "kustomize-test",
			CreateNamespace: true,
			Sources: []manifests.ManifestSource{
				{
					// Example of inline kustomize content
					KustomizeContent: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- https://raw.githubusercontent.com/kubernetes/dashboard/v2.7.0/aio/deploy/recommended.yaml

# Customize namespace
namespace: kustomize-test

# Add labels to all resources
commonLabels:
  app: kubernetes-dashboard
  environment: test

# Configure the Dashboard service
patches:
- target:
    kind: Service
    name: kubernetes-dashboard
  patch: |-
    - op: replace
      path: /spec/type
      value: NodePort
`,
				},
			},
		},
		overrides,
	)
}

// NewInlineKustomizeExample creates a component that demonstrates inline Kustomization content
func NewInlineKustomizeExample(overrides *manifests.ManifestConfig) *ManifestComponent {
	return NewManifestComponent(
		"inline-kustomize-example",
		&manifests.ManifestConfig{
			Namespace:       "kustomize-test",
			CreateNamespace: true,
			Sources: []manifests.ManifestSource{
				{
					// Example of using inline Kustomize content
					KustomizeContent: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

# Define resources through generators, not external files
resources: []

# Apply transformers
transformers:
- |-
  apiVersion: builtin
  kind: NamespaceTransformer
  metadata:
    name: test-namespace-transformer
    namespace: kustomize-test
  setRoleBindingSubjects: defaultOnly

# Define the resource content directly
generatorOptions:
  disableNameSuffixHash: true

configMapGenerator:
- name: kustomize-test-config
  literals:
  - foo=bar
  - hello=world
`,
				},
				{
					// Define the resource.yaml content inline
					Content: `apiVersion: v1
kind: Service
metadata:
  name: test-service
spec:
  selector:
    app: test
  ports:
  - port: 80
    targetPort: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: test-container
        image: nginx:latest
        ports:
        - containerPort: 8080
`,
				},
			},
		},
		overrides,
	)
}

// NewAdvancedKustomizeExample creates a component that demonstrates more complex Kustomization techniques
func NewAdvancedKustomizeExample(overrides *manifests.ManifestConfig) *ManifestComponent {
	return NewManifestComponent(
		"advanced-kustomize-example",
		&manifests.ManifestConfig{
			Namespace:       "kustomize-advanced",
			CreateNamespace: true,
			Sources: []manifests.ManifestSource{
				{
					// Example of inline content for resources to be referenced by kustomize
					Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
spec:
  selector:
    app: nginx
  ports:
  - port: 80
    targetPort: 80
`,
				},
				{
					// Example of using KustomizeContent with a simple configuration
					KustomizeContent: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

# Define an empty resources list - we'll use the separate content source
resources: []

# Define the namespace for all resources
namespace: kustomize-advanced

# Create ConfigMaps from literals
configMapGenerator:
- name: nginx-config
  literals:
  - server_name=example.com
  - worker_processes=auto
  - environment=production

# Add labels to all resources
commonLabels:
  app.kubernetes.io/managed-by: atmosphere
  app.kubernetes.io/component: example
`,
				},
			},
		},
		overrides,
	)
}
