package resources

import (
	"context"
	"io"

	"github.com/ovn-org/libovsdb/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// List is a generic list structure for resources
type List interface {
	runtime.Object
	GetItems() []runtime.Object
}

// ResourceList represents a list of resources with metadata
type ResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
}

// Resource defines the interface for a resource that can be fetched
type Resource interface {
	// Name returns the resource name (e.g., "routers")
	Name() string

	// Aliases returns alternative names for the resource (e.g., ["router"] for "routers")
	Aliases() []string

	// List fetches resources and returns them as a runtime.Object list
	List(ctx context.Context, client client.Client, names []string) (runtime.Object, error)

	// GetTable converts a runtime.Object list to a table representation
	GetTable(obj runtime.Object) (*metav1.Table, error)
}

// Registry holds all registered resources
type Registry struct {
	resources map[string]Resource
}

// NewRegistry creates a new resource registry
func NewRegistry() *Registry {
	return &Registry{
		resources: make(map[string]Resource),
	}
}

// Register adds a resource to the registry
func (r *Registry) Register(resource Resource) {
	r.resources[resource.Name()] = resource

	// Also register aliases
	for _, alias := range resource.Aliases() {
		r.resources[alias] = resource
	}
}

// Get retrieves a resource by name
func (r *Registry) Get(name string) (Resource, bool) {
	resource, ok := r.resources[name]
	return resource, ok
}

// List returns all registered resources
func (r *Registry) List() []string {
	seen := make(map[Resource]bool)
	var names []string

	for name, resource := range r.resources {
		if !seen[resource] && name == resource.Name() {
			names = append(names, name)
			seen[resource] = true
		}
	}

	return names
}

// OVNConfig holds OVN connection configuration
type OVNConfig struct {
	Endpoints []string
	Namespace string

	// For northbound database
	NBStatefulSet string
	NBPort        string

	// For southbound database
	SBStatefulSet string
	SBPort        string
}

// DefaultOVNConfig returns the default OVN configuration
func DefaultOVNConfig() *OVNConfig {
	return &OVNConfig{
		Namespace:     "openstack",
		NBStatefulSet: "ovn-ovsdb-nb",
		NBPort:        "6641",
		SBStatefulSet: "ovn-ovsdb-sb",
		SBPort:        "6642",
	}
}

// GetNBEndpoints returns the northbound database endpoints
func (c *OVNConfig) GetNBEndpoints() []string {
	if len(c.Endpoints) > 0 {
		return c.Endpoints
	}

	// Generate default endpoints
	return []string{
		"tcp:" + c.NBStatefulSet + "-0." + c.NBStatefulSet + "." + c.Namespace + ".svc.cluster.local:" + c.NBPort,
		"tcp:" + c.NBStatefulSet + "-1." + c.NBStatefulSet + "." + c.Namespace + ".svc.cluster.local:" + c.NBPort,
		"tcp:" + c.NBStatefulSet + "-2." + c.NBStatefulSet + "." + c.Namespace + ".svc.cluster.local:" + c.NBPort,
	}
}

// GetSBEndpoints returns the southbound database endpoints
func (c *OVNConfig) GetSBEndpoints() []string {
	if len(c.Endpoints) > 0 {
		return c.Endpoints
	}

	// Generate default endpoints
	return []string{
		"tcp:" + c.SBStatefulSet + "-0." + c.SBStatefulSet + "." + c.Namespace + ".svc.cluster.local:" + c.SBPort,
		"tcp:" + c.SBStatefulSet + "-1." + c.SBStatefulSet + "." + c.Namespace + ".svc.cluster.local:" + c.SBPort,
		"tcp:" + c.SBStatefulSet + "-2." + c.SBStatefulSet + "." + c.Namespace + ".svc.cluster.local:" + c.SBPort,
	}
}

// GetOptions contains common options for get operations
type GetOptions struct {
	ConfigFlags  *genericclioptions.ConfigFlags
	OVNConfig    *OVNConfig
	OutputFormat string
	NoHeaders    bool
	Out          io.Writer
}
