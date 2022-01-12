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
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	ParentSubnetName string `json:"parentSubnetName,omitempty"`
	// NetworkName contains a reference (name) to the network
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	NetworkName string `json:"networkName"`
	// Regions represents the network service location
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Regions []Region `json:"regions"`
	// Consumer refers to resource Subnet has been booked for
	// +kubebuilder:validation:Optional
	Consumer *ResourceReference `json:"consumer,omitempty"`
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

type Region struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-./a-z0-9]*[a-z0-9])?$
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	AvailabilityZones []string `json:"availabilityZones"`
}

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
func (in *Subnet) PopulateStatus() {
	in.Status.State = CProcessingSubnetState

	// Validator checks that slice has at least one element,
	// so it is safe to assume that there is an element on zero index.
	// It is also okay to check AZ count only for first region,
	// since if there is more than one region, it gets classified as
	// multiregion subnet.
	azCount := len(in.Spec.Regions[0].AvailabilityZones)
	regionCount := len(in.Spec.Regions)
	if azCount == 1 && regionCount == 1 {
		in.Status.Locality = CLocalSubnetLocalityType
	} else if azCount > 1 && regionCount == 1 {
		in.Status.Locality = CRegionalSubnetLocalityType
	} else {
		in.Status.Locality = CMultiregionalSubnetLocalityType
	}
}

func (in *Subnet) FillStatusFromCidr(cidr *CIDR) {
	if cidr.IsIPv4() {
		in.Status.Type = CIPv4SubnetType
	} else {
		in.Status.Type = CIPv6SubnetType
	}

	in.Status.Reserved = cidr.DeepCopy()
	in.Status.Vacant = []CIDR{*cidr.DeepCopy()}
	in.Status.PrefixBits = cidr.MaskOnes()
	capacityString := cidr.AddressCapacity().String()
	in.Status.Capacity = resource.MustParse(capacityString)
	in.Status.CapacityLeft = in.Status.Capacity.DeepCopy()
	in.Status.State = CFinishedSubnetState
}

func (in *Subnet) ProposeForCapacity(capacity *resource.Quantity) (*CIDR, error) {
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

	if in.Status.Reserved == nil {
		return nil, errors.New("cidr is not set, can't compute the network prefix")
	}

	maskBits := in.Status.Reserved.MaskBits()

	return in.ProposeForBits(maskBits - byte(bitLen))
}

func (in *Subnet) ProposeForBits(prefixBits byte) (*CIDR, error) {
	if prefixBits > in.Status.Reserved.MaskBits() {
		return nil, errors.New("prefix bit count is bigger than bit coint in IP")
	}

	var candidateOnes byte
	var candidateCidr *CIDR
	for _, cidr := range in.Status.Vacant {
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
func (in *Subnet) Reserve(cidr *CIDR) error {
	var remainingCidrs []CIDR
	reservationIdx := -1
	for i, vacantCidr := range in.Status.Vacant {
		if vacantCidr.CanReserve(cidr) {
			remainingCidrs = vacantCidr.Reserve(cidr)
			reservationIdx = i
			break
		}
	}

	if reservationIdx == -1 {
		return errors.Errorf("unable to find CIDR that includes CIDR %s", cidr.String())
	}

	remainingCidrsCount := len(remainingCidrs)
	switch remainingCidrsCount {
	case 0:
		in.Status.Vacant = append(in.Status.Vacant[:reservationIdx], in.Status.Vacant[reservationIdx+1:]...)
	case 1:
		in.Status.Vacant[reservationIdx] = remainingCidrs[0] // nolint // Ignore linting warning as slice length is checked before (by) switch statement
	default:
		released := make([]CIDR, len(in.Status.Vacant)+remainingCidrsCount-1)
		copy(released[:reservationIdx], in.Status.Vacant[:reservationIdx])
		copy(released[reservationIdx:reservationIdx+remainingCidrsCount], remainingCidrs)
		copy(released[reservationIdx+remainingCidrsCount:], in.Status.Vacant[reservationIdx+1:])
		in.Status.Vacant = released
	}

	in.Status.CapacityLeft.Sub(resource.MustParse(cidr.AddressCapacity().String()))

	return nil
}

// CanReserve checks if it is possible to reserve CIDR
func (in *Subnet) CanReserve(cidr *CIDR) bool {
	for _, vacantCidr := range in.Status.Vacant {
		if vacantCidr.CanReserve(cidr) {
			return true
		}
	}

	return false
}

// Release puts CIDR to vacant range if there are no intersections
// and joins neighbour networks
func (in *Subnet) Release(cidr *CIDR) error {
	if in.Status.Reserved == nil {
		return errors.New("subnet address space hasn't been allocated yet")
	}
	if !in.Status.Reserved.CanReserve(cidr) {
		return errors.Errorf("cidr %s is not describing subnet of %s", cidr.String(), in.Status.Reserved.String())
	}

	vacantLen := len(in.Status.Vacant)
	if vacantLen == 0 {
		in.Status.Vacant = []CIDR{*cidr.DeepCopy()}
		return nil
	}

	insertIdx := -1
	if in.Status.Vacant[0].After(cidr) {
		in.Status.Vacant = append(in.Status.Vacant, CIDR{})
		copy(in.Status.Vacant[1:], in.Status.Vacant)
		in.Status.Vacant[0] = *cidr.DeepCopy()
		insertIdx = 0
	}

	if in.Status.Vacant[vacantLen-1].Before(cidr) {
		in.Status.Vacant = append(in.Status.Vacant, *cidr.DeepCopy())
		insertIdx = vacantLen
	}

	if insertIdx < 0 {
		for idx := 1; idx < vacantLen; idx++ {
			prevIdx := idx - 1
			if in.Status.Vacant[prevIdx].Before(cidr) && in.Status.Vacant[idx].After(cidr) {
				in.Status.Vacant = append(in.Status.Vacant, CIDR{})
				copy(in.Status.Vacant[idx+1:], in.Status.Vacant[idx:])
				in.Status.Vacant[idx] = *cidr.DeepCopy()
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
		if in.Status.Vacant[insertIdx].IsLeft() &&
			insertIdx != len(in.Status.Vacant)-1 {
			potentialJoinIdx = insertIdx + 1
		}

		if in.Status.Vacant[insertIdx].IsRight() &&
			insertIdx != 0 {
			potentialJoinIdx = insertIdx - 1
		}

		if potentialJoinIdx >= 0 &&
			in.Status.Vacant[insertIdx].CanJoin(&in.Status.Vacant[potentialJoinIdx]) {
			in.Status.Vacant[insertIdx].Join(&in.Status.Vacant[potentialJoinIdx])
			in.Status.Vacant = append(in.Status.Vacant[:potentialJoinIdx], in.Status.Vacant[potentialJoinIdx+1:]...)

			if insertIdx > potentialJoinIdx {
				insertIdx = insertIdx - 1
			}
		} else {
			hasMoreJoins = false
		}
	}

	in.Status.CapacityLeft.Add(resource.MustParse(cidr.AddressCapacity().String()))

	return nil
}

// CanRelease checks whether it is possible to release CIDR into current vacant range
func (in *Subnet) CanRelease(cidr *CIDR) bool {
	if in.Status.Reserved == nil {
		return false
	}
	if !in.Status.Reserved.CanReserve(cidr) {
		return false
	}

	vacantLen := len(in.Status.Vacant)
	if vacantLen == 0 {
		return true
	}

	if in.Status.Vacant[0].After(cidr) {
		return true
	}

	if in.Status.Vacant[vacantLen-1].Before(cidr) {
		return true
	}

	for idx := 1; idx < vacantLen; idx++ {
		prevIdx := idx - 1
		if in.Status.Vacant[prevIdx].Before(cidr) && in.Status.Vacant[idx].After(cidr) {
			return true
		}
	}

	return false
}
