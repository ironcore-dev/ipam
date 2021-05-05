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

// SubnetMachineRequestSpec defines the desired state of SubnetMachineRequest
type SubnetMachineRequestSpec struct {
	// Subnet to get IP from
	Subnet string `json:"subnet,omitempty"`
	// MachineRequest for subnet to get the IP
	MachineRequest string `json:"machineRequest,omitempty"`
	// IP to request, if not specified - will be added automatically
	// +kubebuilder:validation:Optional
	IP string `json:"ip,omitempty"`
}

// SubnetMachineRequestStatus defines the observed state of SubnetMachineRequest
type SubnetMachineRequestStatus struct {
	// Status and relevant message for error
	Status  string `json:"status"`
	Message string `json:"message"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SubnetMachineRequest is the Schema for the subnetmachinerequests API
type SubnetMachineRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SubnetMachineRequestSpec   `json:"spec,omitempty"`
	Status SubnetMachineRequestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SubnetMachineRequestList contains a list of SubnetMachineRequest
type SubnetMachineRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SubnetMachineRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SubnetMachineRequest{}, &SubnetMachineRequestList{})
}
