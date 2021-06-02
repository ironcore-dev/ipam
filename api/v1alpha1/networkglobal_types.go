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
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NetworkGlobalSpec defines the desired state of NetworkGlobal
type NetworkGlobalSpec struct {
	// Description contains a human readable description of network
	Description string `json:"description,omitempty"`
}

// NetworkGlobalStatus defines the observed state of NetworkGlobal
type NetworkGlobalStatus struct {
	Ranges   []CIDR            `json:"ranges"`
	Capacity resource.Quantity `json:"capacity,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Capacity",type=string,JSONPath=`.status.capacity`,description="Total address capacity in all ranges"
// +kubebuilder:printcolumn:name="Description",type=string,JSONPath=`.spec.description`,description="Description"
// NetworkGlobal is the Schema for the networkglobals API
type NetworkGlobal struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkGlobalSpec   `json:"spec,omitempty"`
	Status NetworkGlobalStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetworkGlobalList contains a list of NetworkGlobal
type NetworkGlobalList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkGlobal `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkGlobal{}, &NetworkGlobalList{})
}

func (s *NetworkGlobal) Release(cidr *CIDR) error {
	reservationIdx := -1
	for i, reservedCidrs := range s.Status.Ranges {
		if reservedCidrs.Equal(cidr) {
			reservationIdx = i
		}
	}

	if reservationIdx == -1 {
		return errors.Errorf("unable to find CIRD that includes CIDR %s", cidr.String())
	}

	s.Status.Ranges = append(s.Status.Ranges[:reservationIdx], s.Status.Ranges[reservationIdx+1:]...)

	s.Status.Capacity.Sub(resource.MustParse(cidr.AddressCapacity().String()))

	return nil
}

func (s *NetworkGlobal) CanRelease(cidr *CIDR) bool {
	for _, vacantCidr := range s.Status.Ranges {
		if vacantCidr.Equal(cidr) {
			return true
		}
	}

	return false
}

func (s *NetworkGlobal) Reserve(cidr *CIDR) error {
	vacantLen := len(s.Status.Ranges)
	if vacantLen == 0 {
		s.Status.Ranges = []CIDR{*cidr}
		return nil
	}

	insertIdx := -1
	if s.Status.Ranges[0].After(cidr) {
		s.Status.Ranges = append(s.Status.Ranges, CIDR{})
		copy(s.Status.Ranges[1:], s.Status.Ranges)
		s.Status.Ranges[0] = *cidr
		insertIdx = 0
	}

	if s.Status.Ranges[vacantLen-1].Before(cidr) {
		s.Status.Ranges = append(s.Status.Ranges, *cidr)
		insertIdx = vacantLen
	}

	if insertIdx < 0 {
		for idx := 1; idx < vacantLen; idx++ {
			prevIdx := idx - 1
			if s.Status.Ranges[prevIdx].Before(cidr) && s.Status.Ranges[idx].After(cidr) {
				s.Status.Ranges = append(s.Status.Ranges, CIDR{})
				copy(s.Status.Ranges[idx+1:], s.Status.Ranges[idx:])
				s.Status.Ranges[idx] = *cidr
				insertIdx = idx
				break
			}
		}
	}

	if insertIdx < 0 {
		return errors.New("unable to find place to insert cidr")
	}

	if s.Status.Capacity.IsZero() {
		s.Status.Capacity = resource.MustParse(cidr.AddressCapacity().String())
	} else {
		s.Status.Capacity.Add(resource.MustParse(cidr.AddressCapacity().String()))
	}

	return nil
}

func (s *NetworkGlobal) CanReserve(cidr *CIDR) bool {
	vacantLen := len(s.Status.Ranges)
	if vacantLen == 0 {
		return true
	}

	if s.Status.Ranges[0].After(cidr) {
		return true
	}

	if s.Status.Ranges[vacantLen-1].Before(cidr) {
		return true
	}

	for idx := 1; idx < vacantLen; idx++ {
		prevIdx := idx - 1
		if s.Status.Ranges[prevIdx].Before(cidr) && s.Status.Ranges[idx].After(cidr) {
			return true
		}
	}

	return false
}
