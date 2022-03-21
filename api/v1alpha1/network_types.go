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
	// For GENEVE it is a single 24 bit value. First 100 values are reserved.
	// For MLPS it is a set of 20 bit values. First 16 values are reserved.
	// Represented with number encoded to string.
	// +kubebuilder:validation:Optional
	ID *NetworkID `json:"id,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Enum=VXLAN;GENEVE;MPLS
	Type NetworkType `json:"type,omitempty"`
	// Description contains a human readable description of network
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`
}

const (
	CFailedNetworkState     NetworkState = "Failed"
	CProcessingNetworkState NetworkState = "Processing"
	CFinishedNetworkState   NetworkState = "Finished"
)

type NetworkState string

// NetworkStatus defines the observed state of Network
type NetworkStatus struct {
	// IPv4Ranges is a list of IPv4 ranges booked by child subnets
	IPv4Ranges []CIDR `json:"ipv4Ranges,omitempty"`
	// IPv6Ranges is a list of IPv6 ranges booked by child subnets
	IPv6Ranges []CIDR `json:"ipv6Ranges,omitempty"`
	// Reserved is a reserved network ID
	Reserved *NetworkID `json:"reserved,omitempty"`
	// IPv4Capacity is a total address capacity of all IPv4 CIDRs in Ranges
	IPv4Capacity resource.Quantity `json:"ipv4Capacity,omitempty"`
	// IPv6Capacity is a total address capacity of all IPv4 CIDRs in Ranges
	IPv6Capacity resource.Quantity `json:"ipv6Capacity,omitempty"`
	// State is a network creation request processing state
	State NetworkState `json:"state,omitempty"`
	// Message contains error details if the one has occurred
	Message string `json:"message,omitempty"`
}

// Network is the Schema for the networks API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`,description="Network Type"
// +kubebuilder:printcolumn:name="Reserved",type=string,JSONPath=`.status.reserved`,description="Reserved Network ID"
// +kubebuilder:printcolumn:name="IPv4 Capacity",type=string,JSONPath=`.status.ipv4Capacity`,description="Total IPv4 address capacity in all ranges"
// +kubebuilder:printcolumn:name="IPv6 Capacity",type=string,JSONPath=`.status.ipv6Capacity`,description="Total IPv4 address capacity in all ranges"
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

func (in *Network) Release(cidr *CIDR) error {
	ranges := in.getRangesForCidr(cidr)
	reservationIdx := -1

	leftSearchBorder := 0
	rightSearchBorder := len(ranges) - 1
	networkIdx, err := FindParentNetworkIdx(ranges, cidr, leftSearchBorder, rightSearchBorder)
	if err != nil {
		return err
	}
	if ranges[networkIdx].Equal(cidr) {
		reservationIdx = networkIdx
	}

	if reservationIdx == -1 {
		return errors.Errorf("unable to find CIDR that includes CIDR %s", cidr.String())
	}

	ranges = append(ranges[:reservationIdx], ranges[reservationIdx+1:]...)
	sub := resource.MustParse(cidr.AddressCapacity().String())
	in.subCapacityForCidr(cidr, &sub)
	in.setRangesForCidr(cidr, ranges)

	return nil
}

func (in *Network) CanRelease(cidr *CIDR) bool {
	ranges := in.getRangesForCidr(cidr)
	leftSearchBorder := 0
	rightSearchBorder := len(ranges) - 1
	networkIdx, err := FindParentNetworkIdx(ranges, cidr, leftSearchBorder, rightSearchBorder)
	if err != nil {
		return false
	}
	if ranges[networkIdx].Equal(cidr) {
		return true
	}
	return false
}

func FindParentNetworkIdx(ranges []CIDR, cidr *CIDR, left int, right int) (int, error) {
	if len(ranges) == 0 {
		return 0, errors.New("No subnets left")
	}
	theirFirstIP, _ := cidr.ToAddressRange()
	for left < right {
		mid := left + (right-left)/2
		outFirstIP, ourLastIP := ranges[mid].ToAddressRange()
		if outFirstIP.Compare(theirFirstIP) < 0 && ourLastIP.Compare(theirFirstIP) < 0 {
			left = mid + 1
		} else {
			right = mid
		}
	}
	return left, nil
}

func (in *Network) Reserve(cidr *CIDR) error {
	ranges := in.getRangesForCidr(cidr)
	vacantLen := len(ranges)
	if vacantLen == 0 {
		ranges = []CIDR{*cidr}

		add := resource.MustParse(cidr.AddressCapacity().String())
		in.addCapacityForCidr(cidr, &add)
		in.setRangesForCidr(cidr, ranges)

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

	leftSearchBorder := 1
	rightSearchBorder := vacantLen
	networkIdx, err := FindParentNetworkIdx(ranges, cidr, leftSearchBorder, rightSearchBorder)
	if err != nil {
		return err
	}
	if ranges[networkIdx-1].Before(cidr) && ranges[networkIdx].After(cidr) {
		ranges = append(ranges, CIDR{})
		copy(ranges[networkIdx+1:], ranges[networkIdx:])
		ranges[networkIdx] = *cidr
		insertIdx = networkIdx
	}

	if insertIdx < 0 {
		return errors.New("unable to find place to insert cidr")
	}

	add := resource.MustParse(cidr.AddressCapacity().String())
	in.addCapacityForCidr(cidr, &add)
	in.setRangesForCidr(cidr, ranges)

	return nil
}

func (in *Network) CanReserve(cidr *CIDR) bool {
	ranges := in.getRangesForCidr(cidr)
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

func (in *Network) getRangesForCidr(cidr *CIDR) []CIDR {
	if cidr.IsIPv4() {
		return in.Status.IPv4Ranges
	}
	return in.Status.IPv6Ranges
}

func (in *Network) setRangesForCidr(cidr *CIDR, ranges []CIDR) {
	if cidr.IsIPv4() {
		in.Status.IPv4Ranges = ranges
	} else {
		in.Status.IPv6Ranges = ranges
	}
}

func (in *Network) addCapacityForCidr(cidr *CIDR, add *resource.Quantity) {
	if cidr.IsIPv4() {
		in.Status.IPv4Capacity.Add(*add)
	} else {
		in.Status.IPv6Capacity.Add(*add)
	}
}

func (in *Network) subCapacityForCidr(cidr *CIDR, sub *resource.Quantity) {
	if cidr.IsIPv4() {
		in.Status.IPv4Capacity.Sub(*sub)
	} else {
		in.Status.IPv6Capacity.Sub(*sub)
	}
}
