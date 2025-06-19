package components

import (
	"github.com/vexxhost/atmosphere/pkg/helm"
	"github.com/vexxhost/atmosphere/pkg/manifests"
)

func NewVaultOperator(overrides *helm.ComponentConfig) *HelmComponent {
	return &HelmComponent{
		Name: "vault-operator",
		BaseConfig: &helm.ComponentConfig{
			Chart: &helm.ChartConfig{
				Name: "oci://ghcr.io/bank-vaults/helm-charts/vault-operator:1.22.6",
			},
			Release: &helm.ReleaseConfig{
				Namespace: "vault",
				Name:      "vault-operator",
				Values: map[string]interface{}{
					"image": map[string]interface{}{
						"tag": "v1.22.6",
					},
					"bankVaults": map[string]interface{}{
						"image": map[string]interface{}{
							"tag": "v1.31.4",
						},
					},
				},
			},
		},
		Overrides: overrides,
	}
}

func NewVaultOperatorCRDs(overrides *manifests.ManifestConfig) *ManifestComponent {
	return NewManifestComponent(
		"vault-operator-crds",
		&manifests.ManifestConfig{
			Sources: []manifests.ManifestSource{
				{
					URL: "https://raw.githubusercontent.com/bank-vaults/vault-operator/refs/tags/v1.22.6/deploy/charts/vault-operator/crds/crd.yaml",
				},
			},
		},
		overrides,
	)
}

func NewVaultOperatorRBAC(overrides *manifests.ManifestConfig) *ManifestComponent {
	return NewManifestComponent(
		"vault-operator-rbac",
		&manifests.ManifestConfig{
			Namespace: "vault",
			Sources: []manifests.ManifestSource{
				{
					KustomizeContent: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

# Define resources through generators, not external files
resources:
  - https://github.com/bank-vaults/vault-operator/deploy/rbac?ref=v1.22.6

transformers:
- |-
  apiVersion: builtin
  kind: NamespaceTransformer
  metadata:
    name: vault-namespace-transform
    namespace: vault
  setRoleBindingSubjects: defaultOnly

# Define the resource content directly
generatorOptions:
  disableNameSuffixHash: true
`,
				},
			},
		},
		overrides,
	)
}
