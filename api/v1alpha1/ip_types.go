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

// IpSpec defines the desired state of Ip
type IpSpec struct {
	// Subnet to get IP from
	// +kubebuilder:validation:Required
	Subnet string `json:"subnet,omitempty"`
	// CRD find IP for
	// +kubebuilder:validation:Optional
	CRD *CRD `json:"crd,omitempty"`
	// IP to request, if not specified - will be added automatically
	// +kubebuilder:validation:Optional
	IP string `json:"ip,omitempty"`
}

type CRD struct {
	// Kind is CRD Kind for lookup
	GroupVersion string `json:"groupVersion,omitempty"`
	// Kind is CRD Kind for lookup
	Kind string `json:"kind,omitempty"`
	// Name is CRD Name for lookup
	Name string `json:"name,omitempty"`
}

// IpStatus defines the observed state of Ip
type IpStatus struct {
	LastUsedIP string `json:"lastUsedIp,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="IP",type=string,JSONPath=`.spec.ip`,description="IP Address"
// +kubebuilder:printcolumn:name="Subnet",type=string,JSONPath=`.spec.subnet`,description="Subnet"
// +kubebuilder:printcolumn:name="Resource Group",type=string,JSONPath=`.spec.crd.groupVersion`,description="Resource Group"
// +kubebuilder:printcolumn:name="Resource Kind",type=string,JSONPath=`.spec.crd.kind`,description="Resource Kind"
// +kubebuilder:printcolumn:name="Resource Name",type=string,JSONPath=`.spec.crd.name`,description="Resource Name"
// Ip is the Schema for the ips API
type Ip struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IpSpec   `json:"spec,omitempty"`
	Status IpStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IpList contains a list of Ip
type IpList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Ip `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Ip{}, &IpList{})
}
