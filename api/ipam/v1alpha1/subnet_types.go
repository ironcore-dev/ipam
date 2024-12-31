// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"math/big"
	"net/netip"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
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
	ParentSubnet v1.LocalObjectReference `json:"parentSubnet,omitempty"`
	// NetworkName contains a reference (name) to the network
	// +kubebuilder:validation:Required
	Network v1.LocalObjectReference `json:"network"`
	// Regions represents the network service location
	// +kubebuilder:validation:Optional
	Regions []Region `json:"regions,omitempty"`
	// Gateway represents the gateway address for the subnet
	// +kubebuilder:validation:Optional
	Gateway *IPAddr `json:"gateway,omitempty"`
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
// +kubebuilder:printcolumn:name="Parent Subnet",type=string,JSONPath=`.spec.parentSubnet.name`,description="Parent Subnet"
// +kubebuilder:printcolumn:name="Parent Network",type=string,JSONPath=`.spec.network.name`,description="Parent Network"
// +kubebuilder:printcolumn:name="Reserved",type=string,JSONPath=`.status.reserved`,description="Reserved CIDR"
// +kubebuilder:printcolumn:name="Address Type",type=string,JSONPath=`.status.type`,description="Address Type"
// +kubebuilder:printcolumn:name="Locality",type=string,JSONPath=`.status.locality`,description="Locality"
// +kubebuilder:printcolumn:name="Prefix Bits",type=string,JSONPath=`.status.prefixBits`,description="Amount of ones in netmask"
// +kubebuilder:printcolumn:name="Capacity",type=string,JSONPath=`.status.capacity`,description="Capacity"
// +kubebuilder:printcolumn:name="Capacity Left",type=string,JSONPath=`.status.capacityLeft`,description="Capacity Left"
// +kubebuilder:printcolumn:name="Consumer Group",type=string,JSONPath=`.spec.consumer.apiVersion`,description="Consumer Group"
// +kubebuilder:printcolumn:name="Consumer Kind",type=string,JSONPath=`.spec.consumer.kind`,description="Consumer Kind"
// +kubebuilder:printcolumn:name="Consumer Name",type=string,JSONPath=`.spec.consumer.name`,description="Consumer Name"
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`,description="State"
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`,description="Message"
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// Subnet is the Schema for the subnets API
type Subnet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SubnetSpec   `json:"spec,omitempty"`
	Status SubnetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

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
	in.Status.Message = ""

	regionCount := len(in.Spec.Regions)
	if regionCount == 0 {
		return
	}

	// It is okay to check AZ count only for first region,
	// since if there is more than one region, it gets classified as
	// multiregional subnet.
	azCount := len(in.Spec.Regions[0].AvailabilityZones)

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

	in.Status.Message = ""
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
		return nil, errors.New("prefix bit count is bigger than bit count in IP")
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

	ipNet := netip.PrefixFrom(firstIP, int(prefixBits))
	return CIDRFromNet(ipNet), nil
}

// Reserve books CIDR from the range of vacant CIDRs, if possible
func (in *Subnet) Reserve(cidr *CIDR) error {
	var remainingCidrs []CIDR

	leftSearchBorder := 0
	rightSearchBorder := len(in.Status.Vacant) - 1
	networkIdx, err := FindParentNetworkIdx(in.Status.Vacant, cidr, leftSearchBorder, rightSearchBorder)
	if err != nil {
		return errors.Wrap(err, "unable to find parent CIDR")
	}
	if !in.Status.Vacant[networkIdx].CanReserve(cidr) {
		return errors.Errorf("No CIDR found that includes CIDR %s", cidr.String())
	}
	remainingCidrs = in.Status.Vacant[networkIdx].Reserve(cidr)

	remainingCidrsCount := len(remainingCidrs)
	switch remainingCidrsCount {
	case 0:
		in.Status.Vacant = append(in.Status.Vacant[:networkIdx], in.Status.Vacant[networkIdx+1:]...)
	case 1:
		in.Status.Vacant[networkIdx] = remainingCidrs[0] // nolint // Ignore linting warning as slice length is checked before (by) switch statement
	default:
		released := make([]CIDR, len(in.Status.Vacant)+remainingCidrsCount-1)
		copy(released[:networkIdx], in.Status.Vacant[:networkIdx])
		copy(released[networkIdx:networkIdx+remainingCidrsCount], remainingCidrs)
		copy(released[networkIdx+remainingCidrsCount:], in.Status.Vacant[networkIdx+1:])
		in.Status.Vacant = released
	}

	in.Status.CapacityLeft.Sub(resource.MustParse(cidr.AddressCapacity().String()))

	return nil
}

// CanReserve checks if it is possible to reserve CIDR
func (in *Subnet) CanReserve(cidr *CIDR) bool {
	leftSearchBorder := 0
	rightSearchBorder := len(in.Status.Vacant) - 1
	networkIdx, err := FindParentNetworkIdx(in.Status.Vacant, cidr, leftSearchBorder, rightSearchBorder)
	if err != nil || !in.Status.Vacant[networkIdx].CanReserve(cidr) {
		return false
	}
	return true
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

	leftSearchBorder := 1
	rightSearchBorder := vacantLen
	if insertIdx < 0 {
		networkIdx, err := FindParentNetworkIdx(in.Status.Vacant, cidr, leftSearchBorder, rightSearchBorder)
		if err != nil {
			return err
		}
		if in.Status.Vacant[networkIdx-1].Before(cidr) && in.Status.Vacant[networkIdx].After(cidr) {
			in.Status.Vacant = append(in.Status.Vacant, CIDR{})
			copy(in.Status.Vacant[networkIdx+1:], in.Status.Vacant[networkIdx:])
			in.Status.Vacant[networkIdx] = *cidr.DeepCopy()
			insertIdx = networkIdx
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

	leftSearchBorder := 1
	rightSearchBorder := vacantLen
	networkIdx, err := FindParentNetworkIdx(in.Status.Vacant, cidr, leftSearchBorder, rightSearchBorder)
	if err != nil {
		return false
	}
	return in.Status.Vacant[networkIdx-1].Before(cidr) && in.Status.Vacant[networkIdx].After(cidr)
}
