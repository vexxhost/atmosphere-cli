package components

import (
	"github.com/vexxhost/atmosphere/internal/helm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// CertManager represents the cert-manager component
type CertManager struct {
	*BaseComponent
	chartVersion string
}

// NewCertManager creates a new CertManager component
func NewCertManager() *CertManager {
	return &CertManager{
		BaseComponent: NewBaseComponent("cert-manager", "cert-manager"),
		chartVersion:  "1.17.2",
	}
}

// GetHelmRelease returns the Helm release configuration
func (c *CertManager) GetHelmRelease(configFlags *genericclioptions.ConfigFlags) *helm.Release {
	return &helm.Release{
		RESTClientGetter: configFlags,
		RepoURL:          "https://charts.jetstack.io",
		ChartName:        "cert-manager",
		ChartVersion:     c.chartVersion,
		Namespace:        c.namespace,
		Name:             c.name,
		Values:           c.getValues(),
	}
}

// getValues returns the Helm values for cert-manager
func (c *CertManager) getValues() map[string]interface{} {
	return map[string]interface{}{
		"installCRDs": true,
		"global": map[string]interface{}{
			"leaderElection": map[string]interface{}{
				"namespace": c.namespace,
			},
		},
		"featureGates": "AdditionalCertificateOutputFormats=true",
		"volumes": []corev1.Volume{
			{
				Name: "etc-ssl-certs",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/etc/ssl/certs",
					},
				},
			},
		},
		"volumeMounts": []corev1.VolumeMount{
			{
				Name:      "etc-ssl-certs",
				MountPath: "/etc/ssl/certs",
				ReadOnly:  true,
			},
		},
		"nodeSelector": map[string]string{},
		"webhook": map[string]interface{}{
			"extraArgs": []string{
				"--feature-gates=AdditionalCertificateOutputFormats=true",
			},
		},
	}
}