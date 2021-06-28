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
	"math/big"
	"net"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SubnetSpec defines the desired state of Subnet
type SubnetSpec struct {
	// CIDR represents the IP Address Range
	// +kubebuilder:validation:Optional
	CIDR *CIDR `json:"cidr,omitempty"`
	// PrefixBits is an amount of ones zero bits at the beginning of the netmask
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=128
	PrefixBits *byte `json:"prefixBits,omitempty"`
	// Capacity is a desired amount of addresses; will be ceiled to the closest power of 2.
	// +kubebuilder:validation:Optional
	Capacity *resource.Quantity `json:"capacity,omitempty"`
	// ParentSubnetName contains a reference (name) to the parent subent
	// +kubebuilder:validation:Optional
	ParentSubnetName string `json:"parentSubnetName,omitempty"`
	// NetworkName contains a reference (name) to the network
	// +kubebuilder:validation:Required
	NetworkName string `json:"networkName,omitempty"`
	// Regions represents the network service location
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Regions []string `json:"regions,omitempty"`
	// AvailabilityZones represents the locality of the network segment
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	AvailabilityZones []string `json:"availabilityZones,omitempty"`
}

const (
	CIPv6SubnetType SubnetAddressType = "IPv6"
	CIPv4SubnetType SubnetAddressType = "IPv4"

	CLocalSubnetLocalityType         SubnetLocalityType = "Local"
	CRegionalSubnetLocalityType      SubnetLocalityType = "Regional"
	CMultiregionalSubnetLocalityType SubnetLocalityType = "Multiregional"

	CFailedSubnetState     SubnetState = "Failed"
	CProcessingSubnetState SubnetState = "Processing"
	CFinishedSubnetState   SubnetState = "Finished"
)

// SubnetLocalityType is a type of subnet coverage
type SubnetLocalityType string

// SubnetAddressType is a type (version) of IP protocol
type SubnetAddressType string

// SubnetState is a processing state of subnet resource
type SubnetState string

// SubnetStatus defines the observed state of Subnet
type SubnetStatus struct {
	// Type represents whether CIDR is an IPv4 or IPv6
	Type SubnetAddressType `json:"type,omitempty"`
	// Locality represents subnet regional coverated
	Locality SubnetLocalityType `json:"locality,omitempty"`
	// PrefixBits is an amount of ones zero bits at the beginning of the netmask
	PrefixBits byte `json:"prefixBits,omitempty"`
	// Capacity shows total capacity of CIDR
	Capacity resource.Quantity `json:"capacity,omitempty"`
	// CapacityLeft shows remaining capacity (excluding capacity of child subnets)
	CapacityLeft resource.Quantity `json:"capacityLeft,omitempty"`
	// Reserved is a CIDR that was reserved
	Reserved *CIDR `json:"reserved,omitempty"`
	// Vacant shows CIDR ranges available for booking
	Vacant []CIDR `json:"vacant,omitempty"`
	// State represents the cunnet processing state
	State SubnetState `json:"state,omitempty"`
	// Message contains an error string for the failed State
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Parent Subnet",type=string,JSONPath=`.spec.parentSubnetName`,description="Parent Subnet"
// +kubebuilder:printcolumn:name="Parent Network",type=string,JSONPath=`.spec.networkName`,description="Parent Network"
// +kubebuilder:printcolumn:name="Reserved",type=string,JSONPath=`.status.reserved`,description="Reserved CIDR"
// +kubebuilder:printcolumn:name="Address Type",type=string,JSONPath=`.status.type`,description="Address Type"
// +kubebuilder:printcolumn:name="Locality",type=string,JSONPath=`.status.locality`,description="Locality"
// +kubebuilder:printcolumn:name="Prefix Bits",type=string,JSONPath=`.status.prefixBits`,description="Amount of ones in netmask"
// +kubebuilder:printcolumn:name="Capacity",type=string,JSONPath=`.status.capacity`,description="Capacity"
// +kubebuilder:printcolumn:name="Capacity Left",type=string,JSONPath=`.status.capacityLeft`,description="Capacity Left"
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`,description="State"
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`,description="Message"
// Subnet is the Schema for the subnets API
type Subnet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SubnetSpec   `json:"spec,omitempty"`
	Status SubnetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SubnetList contains a list of Subnet
type SubnetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Subnet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Subnet{}, &SubnetList{})
}

// PopulateStatus fills status subresource with default values
func (s *Subnet) PopulateStatus() {
	s.Status.State = CProcessingSubnetState

	azCount := len(s.Spec.AvailabilityZones)
	regionCount := len(s.Spec.Regions)
	if azCount == 1 && regionCount == 1 {
		s.Status.Locality = CLocalSubnetLocalityType
	} else if azCount > 1 && regionCount == 1 {
		s.Status.Locality = CRegionalSubnetLocalityType
	} else {
		s.Status.Locality = CMultiregionalSubnetLocalityType
	}
}

func (s *Subnet) FillStatusFromCidr(cidr *CIDR) {
	if cidr.IsIPv4() {
		s.Status.Type = CIPv4SubnetType
	} else {
		s.Status.Type = CIPv6SubnetType
	}

	s.Status.Reserved = cidr.DeepCopy()
	s.Status.Vacant = []CIDR{*cidr.DeepCopy()}
	s.Status.PrefixBits = cidr.MaskOnes()
	capacityString := cidr.AddressCapacity().String()
	s.Status.Capacity = resource.MustParse(capacityString)
	s.Status.CapacityLeft = s.Status.Capacity.DeepCopy()
	s.Status.State = CFinishedSubnetState
}

func (s *Subnet) ProposeForCapacity(capacity *resource.Quantity) (*CIDR, error) {
	bigCap := capacity.AsDec().UnscaledBig()
	count := big.NewInt(1)

	if bigCap.Cmp(count) < 0 {
		return nil, errors.New("requested capacity is smaller than 1")
	}

	bigCap.Sub(bigCap, count)

	bitLen := bigCap.BitLen()
	count.Lsh(count, uint(bitLen))

	// Check if the value set to capacity is smaller than power of 2
	// otherwise take the next power of 2
	if bigCap.Cmp(count) > 0 {
		bitLen += 1
	}

	if s.Status.Reserved == nil {
		return nil, errors.New("cidr is not set, can't compute the network prefix")
	}

	maskBits := s.Status.Reserved.MaskBits()

	return s.ProposeForBits(maskBits - byte(bitLen))
}

func (s *Subnet) ProposeForBits(prefixBits byte) (*CIDR, error) {
	if prefixBits > s.Status.Reserved.MaskBits() {
		return nil, errors.New("prefix bit count is bigger than bit coint in IP")
	}

	var candidateOnes byte
	var candidateCidr *CIDR
	for _, cidr := range s.Status.Vacant {
		currentOnes := cidr.MaskOnes()

		if currentOnes <= prefixBits {
			if candidateCidr == nil ||
				currentOnes > candidateOnes {
				candidateOnes = currentOnes
				candidateCidr = cidr.DeepCopy()
			}
			if currentOnes == prefixBits {
				break
			}
		}
	}

	if candidateCidr == nil {
		return nil, errors.Errorf("unable to find cidr that will fit /%d network", prefixBits)
	}

	firstIP, _ := candidateCidr.ToAddressRange()
	cidrBits := candidateCidr.MaskBits()

	ipNet := &net.IPNet{
		IP:   firstIP,
		Mask: net.CIDRMask(int(prefixBits), int(cidrBits)),
	}

	return CIDRFromNet(ipNet), nil
}

// Reserve books CIDR from the range of vacant CIDRs, if possible
func (s *Subnet) Reserve(cidr *CIDR) error {
	var remainingCidrs []CIDR
	reservationIdx := -1
	for i, vacantCidr := range s.Status.Vacant {
		if vacantCidr.CanReserve(cidr) {
			remainingCidrs = vacantCidr.Reserve(cidr)
			reservationIdx = i
			break
		}
	}

	if reservationIdx == -1 {
		return errors.Errorf("unable to find CIRD that includes CIDR %s", cidr.String())
	}

	remainingCidrsCount := len(remainingCidrs)
	switch remainingCidrsCount {
	case 0:
		s.Status.Vacant = append(s.Status.Vacant[:reservationIdx], s.Status.Vacant[reservationIdx+1:]...)
	case 1:
		s.Status.Vacant[reservationIdx] = remainingCidrs[0]
	default:
		released := make([]CIDR, len(s.Status.Vacant)+remainingCidrsCount-1)
		copy(released[:reservationIdx], s.Status.Vacant[:reservationIdx])
		copy(released[reservationIdx:reservationIdx+remainingCidrsCount], remainingCidrs)
		copy(released[reservationIdx+remainingCidrsCount:], s.Status.Vacant[reservationIdx+1:])
		s.Status.Vacant = released
	}

	s.Status.CapacityLeft.Sub(resource.MustParse(cidr.AddressCapacity().String()))

	return nil
}

// CanReserve checks if it is possible to reserve CIDR
func (s *Subnet) CanReserve(cidr *CIDR) bool {
	for _, vacantCidr := range s.Status.Vacant {
		if vacantCidr.CanReserve(cidr) {
			return true
		}
	}

	return false
}

// Release puts CIDR to vacant range if there are no intersections
// and joins neighbour networks
func (s *Subnet) Release(cidr *CIDR) error {
	if !s.Spec.CIDR.CanReserve(cidr) {
		return errors.Errorf("cidr %s is not describing subent of %s", cidr.String(), s.Spec.CIDR.String())
	}

	vacantLen := len(s.Status.Vacant)
	if vacantLen == 0 {
		s.Status.Vacant = []CIDR{*cidr.DeepCopy()}
		return nil
	}

	insertIdx := -1
	if s.Status.Vacant[0].After(cidr) {
		s.Status.Vacant = append(s.Status.Vacant, CIDR{})
		copy(s.Status.Vacant[1:], s.Status.Vacant)
		s.Status.Vacant[0] = *cidr.DeepCopy()
		insertIdx = 0
	}

	if s.Status.Vacant[vacantLen-1].Before(cidr) {
		s.Status.Vacant = append(s.Status.Vacant, *cidr.DeepCopy())
		insertIdx = vacantLen
	}

	if insertIdx < 0 {
		for idx := 1; idx < vacantLen; idx++ {
			prevIdx := idx - 1
			if s.Status.Vacant[prevIdx].Before(cidr) && s.Status.Vacant[idx].After(cidr) {
				s.Status.Vacant = append(s.Status.Vacant, CIDR{})
				copy(s.Status.Vacant[idx+1:], s.Status.Vacant[idx:])
				s.Status.Vacant[idx] = *cidr.DeepCopy()
				insertIdx = idx
				break
			}
		}
	}

	if insertIdx < 0 {
		return errors.New("unable to find place to insert cidr")
	}

	hasMoreJoins := true
	for hasMoreJoins {
		potentialJoinIdx := -1
		if s.Status.Vacant[insertIdx].IsLeft() &&
			insertIdx != len(s.Status.Vacant)-1 {
			potentialJoinIdx = insertIdx + 1
		}

		if s.Status.Vacant[insertIdx].IsRight() &&
			insertIdx != 0 {
			potentialJoinIdx = insertIdx - 1
		}

		if potentialJoinIdx >= 0 &&
			s.Status.Vacant[insertIdx].CanJoin(&s.Status.Vacant[potentialJoinIdx]) {
			s.Status.Vacant[insertIdx].Join(&s.Status.Vacant[potentialJoinIdx])
			s.Status.Vacant = append(s.Status.Vacant[:potentialJoinIdx], s.Status.Vacant[potentialJoinIdx+1:]...)

			if insertIdx > potentialJoinIdx {
				insertIdx = insertIdx - 1
			}
		} else {
			hasMoreJoins = false
		}
	}

	s.Status.CapacityLeft.Add(resource.MustParse(cidr.AddressCapacity().String()))

	return nil
}

// CanRelease checks whether it is possible to release CIDR into current vacant range
func (s *Subnet) CanRelease(cidr *CIDR) bool {
	if !s.Spec.CIDR.CanReserve(cidr) {
		return false
	}

	vacantLen := len(s.Status.Vacant)
	if vacantLen == 0 {
		return true
	}

	if s.Status.Vacant[0].After(cidr) {
		return true
	}

	if s.Status.Vacant[vacantLen-1].Before(cidr) {
		return true
	}

	for idx := 1; idx < vacantLen; idx++ {
		prevIdx := idx - 1
		if s.Status.Vacant[prevIdx].Before(cidr) && s.Status.Vacant[idx].After(cidr) {
			return true
		}
	}

	return false
}
