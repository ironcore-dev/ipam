// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"math/big"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkCounterSpec stores the state of assigned IDs for network type.
type NetworkCounterSpec struct {
	// Vacant is a list of unassigned network IDs.
	Vacant []NetworkIDInterval `json:"vacant,omitempty"`
}

// NetworkCounterStatus defines the observed state of NetworkCounter
type NetworkCounterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkCounter is the Schema for the networkcounters API
type NetworkCounter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkCounterSpec   `json:"spec,omitempty"`
	Status NetworkCounterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkCounterList contains a list of NetworkCounter
type NetworkCounterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkCounter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkCounter{}, &NetworkCounterList{})
}

const (
	GENEVENetworkType NetworkType = "GENEVE"
	VXLANNetworkType  NetworkType = "VXLAN"
	MPLSNetworkType   NetworkType = "MPLS"
)

// NetworkType is a type of network id is assigned to.
type NetworkType string

// First 100 addresses (0-99) are reserved
var VXLANFirstAvaliableID = NetworkIDFromBytes([]byte{99 + 1})
var VXLANMaxID = NetworkIDFromBytes([]byte{255, 255, 255})

var GENEVEFirstAvaliableID = VXLANFirstAvaliableID
var GENEVEMaxID = VXLANMaxID

// First 16 addresses (0-15) are reserved
var MPLSFirstAvailableID = NetworkIDFromBytes([]byte{15 + 1})
var Increment = big.NewInt(1)

func NewNetworkCounterSpec(typ NetworkType) *NetworkCounterSpec {
	switch typ {
	case VXLANNetworkType:
		return &NetworkCounterSpec{
			Vacant: []NetworkIDInterval{
				{
					Begin: VXLANFirstAvaliableID,
					// VXLAN ID consists of 24 bits
					End: VXLANMaxID,
				},
			},
		}
	case GENEVENetworkType:
		return &NetworkCounterSpec{
			Vacant: []NetworkIDInterval{
				{
					Begin: GENEVEFirstAvaliableID,
					// GENEVE ID consists of 24 bits
					End: GENEVEMaxID,
				},
			},
		}
	case MPLSNetworkType:
		return &NetworkCounterSpec{
			Vacant: []NetworkIDInterval{
				{
					Begin: MPLSFirstAvailableID,
					// Don't have end here, since MPLS label potentially may be expanded
					// unlimited amount of times with 20 bit blocks
				},
			},
		}
	default:
		return &NetworkCounterSpec{}
	}
}

func (in *NetworkCounterSpec) Propose() (*NetworkID, error) {
	if len(in.Vacant) == 0 {
		return nil, errors.New("no free IDs left")
	}

	return in.Vacant[0].Propose(), nil
}

func (in *NetworkCounterSpec) CanReserve(id *NetworkID) bool {
	for _, released := range in.Vacant {
		if released.Includes(id) {
			return true
		}
	}

	return false
}

func (in *NetworkCounterSpec) Reserve(id *NetworkID) error {
	if id == nil {
		return errors.New("unable to reserve nil ID")
	}

	idx := -1
	var intervals []NetworkIDInterval
	for i, released := range in.Vacant {
		if released.Includes(id) {
			intervals = released.Reserve(id)
			idx = i
			break
		}
	}

	if idx == -1 {
		return errors.New("unable to reserve network id as it is not found in intervals with vacant ids")
	}

	switch len(intervals) {
	case 0:
		in.Vacant = append(in.Vacant[:idx], in.Vacant[idx+1:]...)
	case 1:
		in.Vacant[idx] = intervals[0]
	case 2:
		released := make([]NetworkIDInterval, len(in.Vacant)+1)
		copy(released[:idx], in.Vacant[:idx])
		copy(released[idx:idx+2], intervals)
		copy(released[idx+2:], in.Vacant[idx+1:])
		in.Vacant = released
	}

	return nil
}

func (in *NetworkCounterSpec) Release(id *NetworkID) error {
	// 4 cases:
	// intervals are empty, just insert
	// id is before first interval
	// 		check if it is on border with interval and extend interval if so
	//		otherwise insert before
	// id is after last interval
	// 		check if it is on border with interval and extend interval if so
	//		otherwise insert after
	// id is in between of 2 intervals; find left and right intervals
	//		if interval on border with both, make interval union
	//		if interval is on border with one, extend this one
	//		otherwise insert new interval with exact value between these two
	// if none of these found, means value is in interval, error should be returned

	if in.Vacant == nil {
		in.Vacant = make([]NetworkIDInterval, 0)
	}

	intervalCount := len(in.Vacant)
	if intervalCount == 0 {
		in.Vacant = append(in.Vacant, NetworkIDInterval{
			Exact: id,
		})
		return nil
	}

	if in.Vacant[0].After(id) {
		if !in.Vacant[0].JoinLeft(id) {
			in.Vacant = append(in.Vacant, NetworkIDInterval{})
			copy(in.Vacant[1:], in.Vacant)
			in.Vacant[0] = NetworkIDInterval{
				Exact: id,
			}
		}
		return nil
	}

	if in.Vacant[intervalCount-1].Before(id) {
		if !in.Vacant[intervalCount-1].JoinRight(id) {
			in.Vacant = append(in.Vacant, NetworkIDInterval{
				Exact: id,
			})
		}
		return nil
	}

	beforeIdx := -1
	afterIdx := -1
	for idx := 1; idx < len(in.Vacant); idx++ {
		prevIdx := idx - 1
		if in.Vacant[prevIdx].Before(id) && in.Vacant[idx].After(id) {
			beforeIdx = prevIdx
			afterIdx = idx
			break
		}
	}

	if beforeIdx == -1 {
		return errors.New("unable to find interval that will fit relased value")
	}

	canJoinBefore := in.Vacant[beforeIdx].CanJoinRight(id)
	canJoinAfter := in.Vacant[afterIdx].CanJoinLeft(id)

	if canJoinBefore && canJoinAfter {
		beginId := in.Vacant[beforeIdx].Begin
		endId := in.Vacant[afterIdx].End

		if beginId == nil && in.Vacant[beforeIdx].Exact != nil {
			beginId = in.Vacant[beforeIdx].Exact
		}

		if endId == nil && in.Vacant[beforeIdx].Exact != nil {
			endId = in.Vacant[afterIdx].Exact
		}

		interval := NetworkIDInterval{
			Begin: beginId,
			End:   endId,
		}

		in.Vacant = append(in.Vacant[:beforeIdx], in.Vacant[afterIdx:]...)
		in.Vacant[beforeIdx] = interval

		return nil
	}

	if canJoinBefore {
		in.Vacant[beforeIdx].JoinRight(id)
		return nil
	}

	if canJoinAfter {
		in.Vacant[afterIdx].JoinLeft(id)
		return nil
	}

	in.Vacant = append(in.Vacant, NetworkIDInterval{})
	copy(in.Vacant[afterIdx+1:], in.Vacant[afterIdx:])
	in.Vacant[afterIdx] = NetworkIDInterval{
		Exact: id,
	}

	return nil
}
