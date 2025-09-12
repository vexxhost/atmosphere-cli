// Copyright 2025 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RouterStatus defines the observed state of Router
type RouterStatus struct {
	// ExternalIPs is the list of external IP addresses for the router
	ExternalIPs []string `json:"externalIPs,omitempty"`

	// Agent is the UUID of the agent hosting this router
	Agent string `json:"agent,omitempty"`
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
