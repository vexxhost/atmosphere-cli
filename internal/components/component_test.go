package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vexxhost/atmosphere/pkg/helm"
)

func TestHelmComponent_MergedConfig(t *testing.T) {
	tests := []struct {
		name       string
		baseConfig *helm.ComponentConfig
		overrides  *helm.ComponentConfig
		want       *helm.ComponentConfig
		wantErr    bool
	}{
		{
			name: "no overrides",
			baseConfig: &helm.ComponentConfig{
				Chart: &helm.ChartConfig{
					RepoURL: "https://example.com",
					Name:    "test-chart",
					Version: "1.0.0",
				},
				Release: &helm.ReleaseConfig{
					Namespace: "default",
					Name:      "test-release",
					Values: map[string]interface{}{
						"replicas": 1,
						"image":    "nginx:latest",
					},
				},
			},
			overrides: nil,
			want: &helm.ComponentConfig{
				Chart: &helm.ChartConfig{
					RepoURL: "https://example.com",
					Name:    "test-chart",
					Version: "1.0.0",
				},
				Release: &helm.ReleaseConfig{
					Namespace: "default",
					Name:      "test-release",
					Values: map[string]interface{}{
						"replicas": 1,
						"image":    "nginx:latest",
					},
				},
			},
		},
		{
			name: "override values",
			baseConfig: &helm.ComponentConfig{
				Chart: &helm.ChartConfig{
					RepoURL: "https://example.com",
					Name:    "test-chart",
					Version: "1.0.0",
				},
				Release: &helm.ReleaseConfig{
					Namespace: "default",
					Name:      "test-release",
					Values: map[string]interface{}{
						"replicas": 1,
						"image":    "nginx:latest",
						"resources": map[string]interface{}{
							"requests": map[string]interface{}{
								"cpu":    "100m",
								"memory": "128Mi",
							},
						},
					},
				},
			},
			overrides: &helm.ComponentConfig{
				Release: &helm.ReleaseConfig{
					Values: map[string]interface{}{
						"replicas": 3,
						"resources": map[string]interface{}{
							"requests": map[string]interface{}{
								"cpu": "200m",
							},
						},
					},
				},
			},
			want: &helm.ComponentConfig{
				Chart: &helm.ChartConfig{
					RepoURL: "https://example.com",
					Name:    "test-chart",
					Version: "1.0.0",
				},
				Release: &helm.ReleaseConfig{
					Namespace: "default",
					Name:      "test-release",
					Values: map[string]interface{}{
						"replicas": 3,
						"image":    "nginx:latest",
						"resources": map[string]interface{}{
							"requests": map[string]interface{}{
								"cpu":    "200m",
								"memory": "128Mi",
							},
						},
					},
				},
			},
		},
		{
			name: "override chart version",
			baseConfig: &helm.ComponentConfig{
				Chart: &helm.ChartConfig{
					RepoURL: "https://example.com",
					Name:    "test-chart",
					Version: "1.0.0",
				},
				Release: &helm.ReleaseConfig{
					Namespace: "default",
					Name:      "test-release",
				},
			},
			overrides: &helm.ComponentConfig{
				Chart: &helm.ChartConfig{
					Version: "2.0.0",
				},
			},
			want: &helm.ComponentConfig{
				Chart: &helm.ChartConfig{
					RepoURL: "https://example.com",
					Name:    "test-chart",
					Version: "2.0.0",
				},
				Release: &helm.ReleaseConfig{
					Namespace: "default",
					Name:      "test-release",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HelmComponent{
				Name:       "test-component",
				BaseConfig: tt.baseConfig,
				Overrides:  tt.overrides,
			}

			got, err := h.MergedConfig()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}