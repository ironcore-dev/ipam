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
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	SubnetName string `json:"subnetName"`
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
	// Message contains error details if the one has occurred
	Message string `json:"message,omitempty"`
}

// IP is the Schema for the ips API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="IP",type=string,JSONPath=`.status.reserved`,description="IP Address"
// +kubebuilder:printcolumn:name="Subnet",type=string,JSONPath=`.spec.subnetName`,description="Subnet"
// +kubebuilder:printcolumn:name="Resource Group",type=string,JSONPath=`.spec.resourceReference.apiVersion`,description="Resource Group"
// +kubebuilder:printcolumn:name="Resource Kind",type=string,JSONPath=`.spec.resourceReference.kind`,description="Resource Kind"
// +kubebuilder:printcolumn:name="Resource Name",type=string,JSONPath=`.spec.resourceReference.name`,description="Resource Name"
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`,description="Processing state"
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`,description="Message"
type IP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPSpec   `json:"spec,omitempty"`
	Status IPStatus `json:"status,omitempty"`
}

// IPList contains a list of IP
// +kubebuilder:object:root=true
type IPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IP `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IP{}, &IPList{})
}
