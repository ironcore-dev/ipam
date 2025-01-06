// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
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
	Subnet v1.LocalObjectReference `json:"subnet"`
	// Consumer refers to resource IP has been booked for
	// +kubebuilder:validation:Optional
	Consumer *ResourceReference `json:"consumer,omitempty"`
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
	// Gateway represents the gateway address for the subnet
	Gateway *IPAddr `json:"gateway,omitempty"`
	// Message contains error details if the one has occurred
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="IP",type=string,JSONPath=`.status.reserved`,description="IP Address"
// +kubebuilder:printcolumn:name="Subnet",type=string,JSONPath=`.spec.subnet.name`,description="Subnet"
// +kubebuilder:printcolumn:name="Consumer Group",type=string,JSONPath=`.spec.consumer.apiVersion`,description="Consumer Group"
// +kubebuilder:printcolumn:name="Consumer Kind",type=string,JSONPath=`.spec.consumer.kind`,description="Consumer Kind"
// +kubebuilder:printcolumn:name="Consumer Name",type=string,JSONPath=`.spec.consumer.name`,description="Consumer Name"
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`,description="Processing state"
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`,description="Message"
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// IP is the Schema for the ips API
type IP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPSpec   `json:"spec,omitempty"`
	Status IPStatus `json:"status,omitempty"`
}

// IPList contains a list of IP
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type IPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IP `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IP{}, &IPList{})
}
