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
	// Description contains a human readable description of network
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`
}

const (
	CFailedRequestState     RequestState = "Failed"
	CProcessingRequestState RequestState = "Processing"
	CFinishedRequestState   RequestState = "Finished"
)

type RequestState string

// NetworkStatus defines the observed state of Network
type NetworkStatus struct {
	// Ranges is a list of ranges booked by child subnets
	Ranges []CIDR `json:"ranges,omitempty"`
	// Capacity is a total address capacity of all CIDRs in Ranges
	Capacity resource.Quantity `json:"capacity,omitempty"`
	// State is a network creation request processing state
	State RequestState `json:"state,omitempty"`
	// Message contains error details if the one has occurred
	Message string `json:"message,omitempty"`
}

// Network is the Schema for the networks API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`,description="Network Type"
// +kubebuilder:printcolumn:name="ID",type=string,JSONPath=`.spec.id`,description="Network ID"
// +kubebuilder:printcolumn:name="Capacity",type=string,JSONPath=`.status.capacity`,description="Total address capacity in all ranges"
// +kubebuilder:printcolumn:name="Description",type=string,JSONPath=`.spec.description`,description="Description"
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

func (s *Network) Release(cidr *CIDR) error {
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

func (s *Network) CanRelease(cidr *CIDR) bool {
	for _, vacantCidr := range s.Status.Ranges {
		if vacantCidr.Equal(cidr) {
			return true
		}
	}

	return false
}

func (s *Network) Reserve(cidr *CIDR) error {
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

func (s *Network) CanReserve(cidr *CIDR) bool {
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
