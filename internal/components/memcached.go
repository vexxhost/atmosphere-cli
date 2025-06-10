package components

import (
	"github.com/vexxhost/atmosphere/internal/helm"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Memcached represents the memcached component
type Memcached struct {
	*BaseComponent
	chartVersion string
	chartSource  string
}

// NewMemcached creates a new Memcached component
func NewMemcached() *Memcached {
	return &Memcached{
		BaseComponent: NewBaseComponent("memcached", "openstack"),
		chartVersion:  "7.8.5",
		chartSource:   "oci://registry-1.docker.io/bitnamicharts/memcached",
	}
}

// GetHelmRelease returns the Helm release configuration
func (m *Memcached) GetHelmRelease(configFlags *genericclioptions.ConfigFlags) *helm.Release {
	return &helm.Release{
		RESTClientGetter: configFlags,
		ChartName:        m.chartSource + ":" + m.chartVersion,
		Namespace:        m.namespace,
		Name:             m.name,
		Values:           m.getValues(),
	}
}

// getValues returns the Helm values for memcached
func (m *Memcached) getValues() map[string]interface{} {
	return map[string]interface{}{}
}