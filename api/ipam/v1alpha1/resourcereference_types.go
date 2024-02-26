// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

// ResourceReference allows to refer a resource of particular type at the same namespace
type ResourceReference struct {
	// APIVersion is resource's API group
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-./a-z0-9]*[a-z0-9])?$
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	APIVersion string `json:"apiVersion,omitempty"`
	// Kind is CRD Kind for lookup
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=^[A-Z]([-A-Za-z0-9]*[A-Za-z0-9])?$
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Kind string `json:"kind"`
	// Name is CRD Name for lookup
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`
}
