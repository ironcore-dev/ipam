package v1alpha1

import (
	"errors"
	"math/big"
)

// First 100 addresses (0-99) are reserved
var CVXLANFirstAvaliableID = NetworkIDFromBytes([]byte{99 + 1})
var CVXLANMaxID = NetworkIDFromBytes([]byte{255, 255, 255})

var CGENEVEFirstAvaliableID = CVXLANFirstAvaliableID
var CGENEVEMaxID = CVXLANMaxID

// First 16 addresses (0-15) are reserved
var CMPLSFirstAvailableID = NetworkIDFromBytes([]byte{15 + 1})
var CIncrement = big.NewInt(1)

func (n *NetworkIDInterval) Includes(id *NetworkID) bool {
	if n.Before(id) || n.After(id) {
		return false
	}

	return true
}

func (n *NetworkIDInterval) After(id *NetworkID) bool {
	if id == nil {
		return false
	}

	if n.Begin != nil && n.Begin.Cmp(&id.Int) > 0 {
		return true
	}

	if n.Exact != nil && n.Exact.Cmp(&id.Int) > 0 {
		return true
	}

	return false
}

func (n *NetworkIDInterval) Before(id *NetworkID) bool {
	if id == nil {
		return false
	}

	if n.End != nil && n.End.Cmp(&id.Int) < 0 {
		return true
	}

	if n.Exact != nil && n.Exact.Cmp(&id.Int) < 0 {
		return true
	}

	return false
}

func (n *NetworkIDInterval) CanJoinLeft(id *NetworkID) bool {
	next := &big.Int{}
	next.Add(&id.Int, CIncrement)

	if n.Begin != nil && n.Begin.Cmp(next) == 0 ||
		n.Exact != nil && n.Exact.Cmp(next) == 0 {
		return true
	}

	return false
}

func (n *NetworkIDInterval) CanJoinRight(id *NetworkID) bool {
	prev := &big.Int{}
	prev.Sub(&id.Int, CIncrement)

	if n.End != nil && n.End.Cmp(prev) == 0 ||
		n.Exact != nil && n.Exact.Cmp(prev) == 0 {
		return true
	}

	return false
}

func (n *NetworkIDInterval) JoinLeft(id *NetworkID) bool {
	if !n.CanJoinLeft(id) {
		return false
	}

	n.Begin = id

	if n.Exact != nil {
		n.End = n.Exact
		n.Exact = nil
	}

	return true
}

func (n *NetworkIDInterval) JoinRight(id *NetworkID) bool {
	if !n.CanJoinRight(id) {
		return false
	}

	n.End = id

	if n.Exact != nil {
		n.Begin = n.Exact
		n.Exact = nil
	}

	return true
}

func (n *NetworkIDInterval) Propose() *NetworkID {
	if n.Exact != nil {
		return n.Exact
	}

	if n.Begin != nil {
		return n.Begin
	}

	if n.End != nil {
		return n.End
	}

	// To create the exact value for network interval, set it to 0 byte value.
	return NetworkIDFromBytes([]byte{0})
}

func (n *NetworkIDInterval) Reserve(id *NetworkID) []NetworkIDInterval {
	// if provided id is not in interval, interval is not changed
	if !n.Includes(id) {
		return []NetworkIDInterval{
			*n,
		}
	}

	// First check if this interval has exact value
	// If it present, return empty subset
	if n.Exact != nil && n.Exact.Cmp(&id.Int) == 0 {
		return []NetworkIDInterval{}
	}

	var newBegin *big.Int
	var newEnd *big.Int

	// Second check is for border cases
	if n.Begin != nil && n.Begin.Cmp(&id.Int) == 0 {
		newBegin = &big.Int{}
		newBegin.Add(&n.Begin.Int, CIncrement)
	}

	if n.End != nil && n.End.Cmp(&id.Int) == 0 {
		newEnd = &big.Int{}
		newEnd.Sub(&n.End.Int, CIncrement)
	}

	// If id is not on border, it is inside of interval
	if newBegin == nil && newEnd == nil {
		newEnd = &big.Int{}
		newEnd.Sub(&id.Int, CIncrement)

		newBegin = &big.Int{}
		newBegin.Add(&id.Int, CIncrement)
	}

	intervals := make([]NetworkIDInterval, 0)

	if newEnd != nil {
		if n.Begin != nil && n.Begin.Cmp(newEnd) == 0 {
			intervals = append(intervals, NetworkIDInterval{
				Exact: NetworkIDFromBigInt(newEnd),
			})
		} else {
			intervals = append(intervals, NetworkIDInterval{
				Begin: n.Begin,
				End:   NetworkIDFromBigInt(newEnd),
			})
		}
	}

	if newBegin != nil {
		if n.End != nil && n.End.Cmp(newBegin) == 0 {
			intervals = append(intervals, NetworkIDInterval{
				Exact: NetworkIDFromBigInt(newBegin),
			})
		} else {
			intervals = append(intervals, NetworkIDInterval{
				Begin: NetworkIDFromBigInt(newBegin),
				End:   n.End,
			})
		}
	}

	return intervals
}

func NewNetworkCounterSpec(typ NetworkType) *NetworkCounterSpec {
	switch typ {
	case CVXLANNetworkType:
		return &NetworkCounterSpec{
			Vacant: []NetworkIDInterval{
				{
					Begin: CVXLANFirstAvaliableID,
					// VXLAN ID consists of 24 bits
					End: CVXLANMaxID,
				},
			},
		}
	case CGENEVENetworkType:
		return &NetworkCounterSpec{
			Vacant: []NetworkIDInterval{
				{
					Begin: CGENEVEFirstAvaliableID,
					// GENEVE ID consists of 24 bits
					End: CGENEVEMaxID,
				},
			},
		}
	case CMPLSNetworkType:
		return &NetworkCounterSpec{
			Vacant: []NetworkIDInterval{
				{
					Begin: CMPLSFirstAvailableID,
					// Don't have end here, since MPLS label potentially may be expanded
					// unlimited amount of times with 20 bit blocks
				},
			},
		}
	default:
		return &NetworkCounterSpec{}
	}
}

func (n *NetworkCounterSpec) Propose() (*NetworkID, error) {
	if len(n.Vacant) == 0 {
		return nil, errors.New("no free IDs left")
	}

	return n.Vacant[0].Propose(), nil
}

func (n *NetworkCounterSpec) CanReserve(id *NetworkID) bool {
	for _, released := range n.Vacant {
		if released.Includes(id) {
			return true
		}
	}

	return false
}

func (n *NetworkCounterSpec) Reserve(id *NetworkID) error {
	if id == nil {
		return errors.New("unable to reserve nil ID")
	}

	idx := -1
	var intervals []NetworkIDInterval
	for i, released := range n.Vacant {
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
		n.Vacant = append(n.Vacant[:idx], n.Vacant[idx+1:]...)
	case 1:
		n.Vacant[idx] = intervals[0]
	case 2:
		released := make([]NetworkIDInterval, len(n.Vacant)+1)
		copy(released[:idx], n.Vacant[:idx])
		copy(released[idx:idx+2], intervals)
		copy(released[idx+2:], n.Vacant[idx+1:])
		n.Vacant = released
	}

	return nil
}

func (n *NetworkCounterSpec) Release(id *NetworkID) error {
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

	if n.Vacant == nil {
		n.Vacant = make([]NetworkIDInterval, 0)
	}

	intervalCount := len(n.Vacant)
	if intervalCount == 0 {
		n.Vacant = append(n.Vacant, NetworkIDInterval{
			Exact: id,
		})
		return nil
	}

	if n.Vacant[0].After(id) {
		if !n.Vacant[0].JoinLeft(id) {
			n.Vacant = append(n.Vacant, NetworkIDInterval{})
			copy(n.Vacant[1:], n.Vacant)
			n.Vacant[0] = NetworkIDInterval{
				Exact: id,
			}
		}
		return nil
	}

	if n.Vacant[intervalCount-1].Before(id) {
		if !n.Vacant[intervalCount-1].JoinRight(id) {
			n.Vacant = append(n.Vacant, NetworkIDInterval{
				Exact: id,
			})
		}
		return nil
	}

	beforeIdx := -1
	afterIdx := -1
	for idx := 1; idx < len(n.Vacant); idx++ {
		prevIdx := idx - 1
		if n.Vacant[prevIdx].Before(id) && n.Vacant[idx].After(id) {
			beforeIdx = prevIdx
			afterIdx = idx
			break
		}
	}

	if beforeIdx == -1 {
		return errors.New("unable to find interval that will fit relased value")
	}

	canJoinBefore := n.Vacant[beforeIdx].CanJoinRight(id)
	canJoinAfter := n.Vacant[afterIdx].CanJoinLeft(id)

	if canJoinBefore && canJoinAfter {
		interval := NetworkIDInterval{
			Begin: n.Vacant[beforeIdx].Begin,
			End:   n.Vacant[afterIdx].End,
		}

		n.Vacant = append(n.Vacant[:beforeIdx], n.Vacant[afterIdx:]...)
		n.Vacant[beforeIdx] = interval

		return nil
	}

	if canJoinBefore {
		n.Vacant[beforeIdx].JoinRight(id)
		return nil
	}

	if canJoinAfter {
		n.Vacant[afterIdx].JoinLeft(id)
		return nil
	}

	n.Vacant = append(n.Vacant, NetworkIDInterval{})
	copy(n.Vacant[afterIdx+1:], n.Vacant[afterIdx:])
	n.Vacant[afterIdx] = NetworkIDInterval{
		Exact: id,
	}

	return nil
}
