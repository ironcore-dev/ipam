/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CFailedIPState     IPState = "Failed"
	CProcessingIPState IPState = "Processing"
	CFinishedIPState   IPState = "Finished"
)

// IPState is a processing state of IP resource
type IPState string

// IPSpec defines the desired state of IP
type IPSpec struct {
	// SubnetName is referring to parent subnet that holds requested IP
	// +kubebuilder:validation:Required
	SubnetName string `json:"subnetName,omitempty"`
	// ResourceReference refers to resource IP has been booked for
	// +kubebuilder:validation:Optional
	ResourceReference *ResourceReference `json:"resourceReference,omitempty"`
	// IP allows to set desired IP address explicitly
	// +kubebuilder:validation:Optional
	IP *IPAddr `json:"ip,omitempty"`
}

// IPStatus defines the observed state of IP
type IPStatus struct {
	// State is a network creation request processing state
	State IPState `json:"state,omitempty"`
	// Reserved is a reserved IP
	Reserved *IPAddr `json:"reserved,omitempty"`
	// Message contains error details if the one has occurred
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="IP",type=string,JSONPath=`.status.reserved`,description="IP Address"
// +kubebuilder:printcolumn:name="Subnet",type=string,JSONPath=`.spec.subnetName`,description="Subnet"
// +kubebuilder:printcolumn:name="Resource Group",type=string,JSONPath=`.spec.resourceReference.apiVersion`,description="Resource Group"
// +kubebuilder:printcolumn:name="Resource Kind",type=string,JSONPath=`.spec.resourceReference.kind`,description="Resource Kind"
// +kubebuilder:printcolumn:name="Resource Name",type=string,JSONPath=`.spec.resourceReference.name`,description="Resource Name"
// IP is the Schema for the ips API
type IP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPSpec   `json:"spec,omitempty"`
	Status IPStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IPList contains a list of IP
type IPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IP `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IP{}, &IPList{})
}

type ResourceReference struct {
	// APIVersion is resource's API group
	// +kubebuilder:validation:Optional
	APIVersion string `json:"apiVersion,omitempty"`
	// Kind is CRD Kind for lookup
	// +kubebuilder:validation:Required
	Kind string `json:"kind,omitempty"`
	// Name is CRD Name for lookup
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
}
