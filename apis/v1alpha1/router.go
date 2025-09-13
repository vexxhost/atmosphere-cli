// Copyright 2025 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// RouterPortInfo defines information about a router port
type RouterPortInfo struct {
	// UUID is the UUID of the logical router port
	UUID types.UID `json:"uuid,omitempty"`

	// InternalUUID is the UUID of the internal port (if any)
	InternalUUID *types.UID `json:"internalUUID,omitempty"`

	// IsGateway indicates if this port is a gateway port
	IsGateway bool `json:"isGateway,omitempty"`
}

// RouterStatus defines the observed state of Router
type RouterStatus struct {
	// Agent is the UUID of the agent hosting this router
	Agent string `json:"agent,omitempty"`

	// ExternalIPs is the list of external IP addresses for the router
	ExternalIPs []string `json:"externalIPs,omitempty"`

	// InternalUUID is the internal UUID of the router (if any)
	InternalUUID *types.UID `json:"internalUUID,omitempty"`

	// Ports is the list of port UUIDs associated with this router
	Ports []RouterPortInfo `json:"ports,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Router represents an OVN router
type Router struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status RouterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RouterList contains a list of Router
type RouterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Router `json:"items"`
}
