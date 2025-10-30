// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"math/big"
)

// NetworkIDInterval represents inclusive interval for network IDs.
// Used to represent intervals of unassigned IDs.
type NetworkIDInterval struct {
	// Begin is a first available value in interval
	Begin *NetworkID `json:"begin,omitempty"`
	// Exact represents a single value in interval
	Exact *NetworkID `json:"exact,omitempty"`
	// End is a last available value in interval
	End *NetworkID `json:"end,omitempty"`
}

func (in *NetworkIDInterval) Includes(id *NetworkID) bool {
	if in.Before(id) || in.After(id) {
		return false
	}

	return true
}

func (in *NetworkIDInterval) After(id *NetworkID) bool {
	if id == nil {
		return false
	}

	if in.Begin != nil && in.Begin.Cmp(&id.Int) > 0 {
		return true
	}

	if in.Exact != nil && in.Exact.Cmp(&id.Int) > 0 {
		return true
	}

	return false
}

func (in *NetworkIDInterval) Before(id *NetworkID) bool {
	if id == nil {
		return false
	}

	if in.End != nil && in.End.Cmp(&id.Int) < 0 {
		return true
	}

	if in.Exact != nil && in.Exact.Cmp(&id.Int) < 0 {
		return true
	}

	return false
}

func (in *NetworkIDInterval) CanJoinLeft(id *NetworkID) bool {
	next := &big.Int{}
	next.Add(&id.Int, Increment)

	if in.Begin != nil && in.Begin.Cmp(next) == 0 ||
		in.Exact != nil && in.Exact.Cmp(next) == 0 {
		return true
	}

	return false
}

func (in *NetworkIDInterval) CanJoinRight(id *NetworkID) bool {
	prev := &big.Int{}
	prev.Sub(&id.Int, Increment)

	if in.End != nil && in.End.Cmp(prev) == 0 ||
		in.Exact != nil && in.Exact.Cmp(prev) == 0 {
		return true
	}

	return false
}

func (in *NetworkIDInterval) JoinLeft(id *NetworkID) bool {
	if !in.CanJoinLeft(id) {
		return false
	}

	in.Begin = id

	if in.Exact != nil {
		in.End = in.Exact
		in.Exact = nil
	}

	return true
}

func (in *NetworkIDInterval) JoinRight(id *NetworkID) bool {
	if !in.CanJoinRight(id) {
		return false
	}

	in.End = id

	if in.Exact != nil {
		in.Begin = in.Exact
		in.Exact = nil
	}

	return true
}

func (in *NetworkIDInterval) Propose() *NetworkID {
	if in.Exact != nil {
		return in.Exact
	}

	if in.Begin != nil {
		return in.Begin
	}

	if in.End != nil {
		return in.End
	}

	// To create the exact value for network interval, set it to 0 byte value.
	return NetworkIDFromBytes([]byte{0})
}

func (in *NetworkIDInterval) Reserve(id *NetworkID) []NetworkIDInterval {
	// if provided id is not in interval, interval is not changed
	if !in.Includes(id) {
		return []NetworkIDInterval{
			*in,
		}
	}

	// First check if this interval has exact value
	// If it present, return empty subset
	if in.Exact != nil && in.Exact.Cmp(&id.Int) == 0 {
		return []NetworkIDInterval{}
	}

	var newBegin *big.Int
	var newEnd *big.Int

	// Second check is for border cases
	if in.Begin != nil && in.Begin.Cmp(&id.Int) == 0 {
		newBegin = &big.Int{}
		newBegin.Add(&in.Begin.Int, Increment)
	}

	if in.End != nil && in.End.Cmp(&id.Int) == 0 {
		newEnd = &big.Int{}
		newEnd.Sub(&in.End.Int, Increment)
	}

	// If id is not on border, it is inside of interval
	if newBegin == nil && newEnd == nil {
		newEnd = &big.Int{}
		newEnd.Sub(&id.Int, Increment)

		newBegin = &big.Int{}
		newBegin.Add(&id.Int, Increment)
	}

	intervals := make([]NetworkIDInterval, 0)

	if newEnd != nil {
		if in.Begin != nil && in.Begin.Cmp(newEnd) == 0 {
			intervals = append(intervals, NetworkIDInterval{
				Exact: NetworkIDFromBigInt(newEnd),
			})
		} else {
			intervals = append(intervals, NetworkIDInterval{
				Begin: in.Begin,
				End:   NetworkIDFromBigInt(newEnd),
			})
		}
	}

	if newBegin != nil {
		if in.End != nil && in.End.Cmp(newBegin) == 0 {
			intervals = append(intervals, NetworkIDInterval{
				Exact: NetworkIDFromBigInt(newBegin),
			})
		} else {
			intervals = append(intervals, NetworkIDInterval{
				Begin: NetworkIDFromBigInt(newBegin),
				End:   in.End,
			})
		}
	}

	return intervals
}
