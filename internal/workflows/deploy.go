package workflows

import (
	flow "github.com/noneback/go-taskflow"
	"github.com/vexxhost/atmosphere/internal/helm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// CreateDeployWorkflow creates and returns a deployment TaskFlow
func CreateDeployWorkflow(configFlags *genericclioptions.ConfigFlags) *flow.TaskFlow {
	// Create a new custom flow
	tf := NewTaskFlow()

	// Create tasks for each helm chart
	// Examples of different chart sources:
	// Local: "./charts/memcached"
	// OCI: "oci://registry.example.com/charts/memcached"
	// Repository: "bitnami/memcached"

	// Example from local path
	// tf.NewDeployHelmChartFlow(&helm.Release{
	// 	RESTClientGetter: configFlags,
	// 	ChartName:    "./charts/memcached",
	// 	Namespace:    "openstack",
	// 	Name:         "memcached",
	// })

	// Example from repo URL
	tf.NewDeployHelmChartFlow(&helm.Release{
		RESTClientGetter: configFlags,
		RepoURL:          "https://kubernetes-sigs.github.io/metrics-server",
		ChartName:        "metrics-server",
		ChartVersion:     "3.12.2",
		Namespace:        "kube-system",
		Name:             "metrics-server",
		Values: map[string]interface{}{
			"args": []string{
				"--kubelet-insecure-tls",
			},
		},
	})

	tf.NewDeployHelmChartFlow(&helm.Release{
		RESTClientGetter: configFlags,
		RepoURL:          "https://charts.jetstack.io",
		ChartName:        "cert-manager",
		ChartVersion:     "1.17.2",
		Namespace:        "cert-manager",
		Name:             "cert-manager",
		Values: map[string]interface{}{
			"installCRDs": true,
			"global": map[string]interface{}{
				"leaderElection": map[string]interface{}{
					"namespace": "cert-manager",
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
		},
	})

	// Example from OCI	registry
	// tf.NewDeployHelmChartFlow(&helm.Release{
	// 	RESTClientGetter: configFlags,
	// 	ChartName:        "oci://registry-1.docker.io/bitnamicharts/memcached:7.8.5",
	// 	Namespace:        "openstack",
	// 	Name:             "memcached",
	// })

	// taskB := tf.NewDeployHelmChartTask("./charts/b", actionConfig)
	// taskC := tf.NewDeployHelmChartTask("./charts/c", actionConfig)

	// Set up dependencies (example: a precedes both b and c)
	// taskA.Precede(taskB, taskC)

	// Return the underlying TaskFlow for execution
	return tf.TaskFlow
}
