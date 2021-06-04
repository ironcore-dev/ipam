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

// NetworkSpec defines the desired state of Network
type NetworkSpec struct {
	// ID is a unique network identifier.
	// For VXLAN it is a single 24 bit value. First 100 values are reserved.
	// For MLPS it is a set of 20 bit values. First 16 values are reserved.
	// Represented with number encoded to string.
	// +kubebuilder:validation:Optional
	ID *NetworkID `json:"id,omitempty"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Enum=VXLAN;MPLS
	Type NetworkType `json:"type,omitempty"`
}

const (
	CFailedRequestState     RequestState = "Failed"
	CProcessingRequestState RequestState = "Processing"
	CFinishedRequestState   RequestState = "Finished"
)

type RequestState string

// NetworkStatus defines the observed state of Network
type NetworkStatus struct {
	State   RequestState `json:"state,omitempty"`
	Message string       `json:"message,omitempty"`
}

// Network is the Schema for the networks API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`,description="Network Type"
// +kubebuilder:printcolumn:name="ID",type=string,JSONPath=`.spec.id`,description="Network ID"
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`,description="Request state"
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`,description="Message about request processing resutls"
type Network struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkSpec   `json:"spec,omitempty"`
	Status NetworkStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetworkList contains a list of Network
type NetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Network `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Network{}, &NetworkList{})
}
