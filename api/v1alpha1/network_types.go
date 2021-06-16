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
	// +kubebuilder:validation:Optional
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
	// IPv4Ranges is a list of IPv4 ranges booked by child subnets
	IPv4Ranges []CIDR `json:"ipv4Ranges,omitempty"`
	// IPv6Ranges is a list of IPv6 ranges booked by child subnets
	IPv6Ranges []CIDR `json:"ipv6Ranges,omitempty"`
	// Reserved is a reserved network ID
	Reserved *NetworkID `json:"reserved,omitempty"`
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
// +kubebuilder:printcolumn:name="Reserved",type=string,JSONPath=`.status.reserved`,description="Reserved Network ID"
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
	ranges := s.getRangesForCidr(cidr)
	reservationIdx := -1
	for i, reservedCidrs := range ranges {
		if reservedCidrs.Equal(cidr) {
			reservationIdx = i
		}
	}

	if reservationIdx == -1 {
		return errors.Errorf("unable to find CIRD that includes CIDR %s", cidr.String())
	}

	ranges = append(ranges[:reservationIdx], ranges[reservationIdx+1:]...)
	s.setRangesForCidr(cidr, ranges)
	s.Status.Capacity.Sub(resource.MustParse(cidr.AddressCapacity().String()))

	return nil
}

func (s *Network) CanRelease(cidr *CIDR) bool {
	ranges := s.getRangesForCidr(cidr)
	for _, vacantCidr := range ranges {
		if vacantCidr.Equal(cidr) {
			return true
		}
	}

	return false
}

func (s *Network) Reserve(cidr *CIDR) error {
	ranges := s.getRangesForCidr(cidr)
	vacantLen := len(ranges)
	if vacantLen == 0 {
		ranges = []CIDR{*cidr}
		s.setRangesForCidr(cidr, ranges)
		return nil
	}

	insertIdx := -1
	if ranges[0].After(cidr) {
		ranges = append(ranges, CIDR{})
		copy(ranges[1:], ranges)
		ranges[0] = *cidr
		insertIdx = 0
	}

	if ranges[vacantLen-1].Before(cidr) {
		ranges = append(ranges, *cidr)
		insertIdx = vacantLen
	}

	if insertIdx < 0 {
		for idx := 1; idx < vacantLen; idx++ {
			prevIdx := idx - 1
			if ranges[prevIdx].Before(cidr) && ranges[idx].After(cidr) {
				ranges = append(ranges, CIDR{})
				copy(ranges[idx+1:], ranges[idx:])
				ranges[idx] = *cidr
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

	s.setRangesForCidr(cidr, ranges)

	return nil
}

func (s *Network) CanReserve(cidr *CIDR) bool {
	ranges := s.getRangesForCidr(cidr)
	vacantLen := len(ranges)
	if vacantLen == 0 {
		return true
	}

	if ranges[0].After(cidr) {
		return true
	}

	if ranges[vacantLen-1].Before(cidr) {
		return true
	}

	for idx := 1; idx < vacantLen; idx++ {
		prevIdx := idx - 1
		if ranges[prevIdx].Before(cidr) && ranges[idx].After(cidr) {
			return true
		}
	}

	return false
}

func (s *Network) getRangesForCidr(cidr *CIDR) []CIDR {
	if cidr.IsIPv4() {
		return s.Status.IPv4Ranges
	}
	return s.Status.IPv6Ranges
}

func (s *Network) setRangesForCidr(cidr *CIDR, ranges []CIDR) {
	if cidr.IsIPv4() {
		s.Status.IPv4Ranges = ranges
	} else {
		s.Status.IPv6Ranges = ranges
	}
}
