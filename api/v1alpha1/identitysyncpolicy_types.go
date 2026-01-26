// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IdentitySyncPolicySpec defines the desired state of IdentitySyncPolicy
type IdentitySyncPolicySpec struct {
	// targetNamespaces is the list of namespaces to sync into.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=50
	// +kubebuilder:validation:Items:MinLength=1
	// +kubebuilder:validation:Items:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +listType=set
	TargetNamespaces []string `json:"targetNamespaces"`

	ServiceAccount ServiceAccount `json:"serviceAccount"`
	Secret         Secret         `json:"secret"`
}

type ServiceAccount struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	Name string `json:"name"`
}

type Secret struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	Name string `json:"name"`

	SourceRef NamespacedNameRef `json:"sourceRef"`
}

type NamespacedNameRef struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	Name string `json:"name"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	Namespace string `json:"namespace"`
}

// IdentitySyncPolicyStatus defines the observed state of IdentitySyncPolicy.
type IdentitySyncPolicyStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedSourceSecretHash is a hash of the last successfully applied source Secret data.
	ObservedSourceSecretHash string `json:"observedSourceSecretHash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// IdentitySyncPolicy is the Schema for the identitysyncpolicies API
// +kubebuilder:resource:scope=Cluster
type IdentitySyncPolicy struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of IdentitySyncPolicy
	// +required
	Spec IdentitySyncPolicySpec `json:"spec"`

	// status defines the observed state of IdentitySyncPolicy
	// +optional
	Status IdentitySyncPolicyStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// IdentitySyncPolicyList contains a list of IdentitySyncPolicy
type IdentitySyncPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []IdentitySyncPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IdentitySyncPolicy{}, &IdentitySyncPolicyList{})
}
