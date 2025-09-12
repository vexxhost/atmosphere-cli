package resources

import (
	"github.com/vexxhost/atmosphere/internal/ovnrouter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RouterInfo contains router information with additional computed fields
type RouterInfo struct {
	ovnrouter.Router
	ExternalIPs []string `json:"externalIPs,omitempty"`
}

// RouterList represents a list of routers
type RouterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RouterInfo `json:"items"`
}

// GetObjectKind returns the object kind
func (r *RouterList) GetObjectKind() schema.ObjectKind {
	return &r.TypeMeta
}

// DeepCopyObject creates a deep copy of the RouterList
func (r *RouterList) DeepCopyObject() runtime.Object {
	// Simple implementation - in production you'd want proper deep copy
	return &RouterList{
		TypeMeta: r.TypeMeta,
		ListMeta: r.ListMeta,
		Items:    append([]RouterInfo(nil), r.Items...),
	}
}