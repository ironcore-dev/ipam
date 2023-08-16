// Copyright 2023 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"encoding/json"
	"math/big"
	"net/netip"

	"go4.org/netipx"
)

// +kubebuilder:validation:Type=string
type CIDR struct {
	Net     netip.Prefix   `json:"-"`
	IPRange netipx.IPRange `json:"-"`
}

func CIDRFromString(cidrString string) (*CIDR, error) {
	cidr, err := netip.ParsePrefix(cidrString)
	if err != nil {
		return nil, err
	}
	ipRange := netipx.RangeOfPrefix(cidr)
	return &CIDR{
		Net:     cidr,
		IPRange: ipRange,
	}, nil
}

func CIDRFromNet(n netip.Prefix) *CIDR {
	ipRange := netipx.RangeOfPrefix(n)
	return &CIDR{
		Net:     n,
		IPRange: ipRange,
	}
}

func (in *CIDR) MarshalJSON() ([]byte, error) {
	return json.Marshal(in.String())
}

func (in *CIDR) UnmarshalJSON(b []byte) error {
	stringVal := string(b)
	if stringVal == "null" {
		return nil
	}
	if err := json.Unmarshal(b, &stringVal); err != nil {
		return err
	}
	pIP, err := netip.ParsePrefix(stringVal)
	if err != nil {
		return err
	}
	in.Net = pIP
	return nil
}

func (in *CIDR) MaskBits() byte {
	return byte(in.Net.Addr().BitLen())
}

func (in *CIDR) MaskOnes() byte {
	return byte(in.Net.Bits())
}

func (in *CIDR) MaskZeroes() byte {
	ones := in.Net.Bits()
	bits := in.Net.Addr().BitLen()
	return byte(bits - ones)
}

func (in *CIDR) AddressCapacity() *big.Int {
	ones := in.Net.Bits()
	bits := in.Net.Addr().BitLen()

	ac := big.Int{}
	ac.Exp(big.NewInt(2), big.NewInt(int64(bits-ones)), nil)

	return &ac
}

func (in *CIDR) MaskCapacity() *big.Int {
	count := in.AddressCapacity()
	count.Sub(count, big.NewInt(1))

	return count
}

func (in *CIDR) ToAddressRange() (netip.Addr, netip.Addr) {
	return in.IPRange.From(), in.IPRange.To()
}

func (in *CIDR) Equal(cidr *CIDR) bool {
	ourOnes := in.Net.Bits()
	ourBits := in.Net.Addr().BitLen()

	theirOnes := cidr.Net.Bits()
	theirBits := cidr.Net.Addr().BitLen()
	firstOurIP, _ := in.ToAddressRange()
	firstTheirIP, _ := cidr.ToAddressRange()

	return firstOurIP.Compare(firstTheirIP) == 0 &&
		ourBits == theirBits &&
		ourOnes == theirOnes
}

func (in *CIDR) IsLeft() bool {
	ones := in.Net.Bits()
	bits := in.Net.Addr().BitLen()
	if ones == 0 {
		return false
	}
	return in.isLeft(ones, bits)
}

func (in *CIDR) isLeft(ones, bits int) bool {
	var ipBytes []byte
	if in.Net.Addr().Is4() {
		ipv4 := in.Net.Addr().As4()
		ipBytes = ipv4[:]
	} else {
		ipv6 := in.Net.Addr().As16()
		ipBytes = ipv6[:]
	}
	ipLen := len(ipBytes)
	bitsDiff := bits - ones
	ipIdx := ipLen - bitsDiff/8 - 1
	ipBit := bitsDiff % 8
	return ipBytes[ipIdx]&(1<<ipBit) == 0
}

func (in *CIDR) IsRight() bool {
	ones := in.Net.Bits()
	bits := in.Net.Addr().BitLen()
	if ones == 0 {
		return false
	}
	return !in.isLeft(ones, bits)
}

func (in *CIDR) Before(cidr *CIDR) bool {
	_, lastIP := in.ToAddressRange()
	firstOtherIP, _ := cidr.ToAddressRange()
	return lastIP.Compare(firstOtherIP) < 0
}

func (in *CIDR) After(cidr *CIDR) bool {
	ourFirstIP, _ := in.ToAddressRange()
	_, lastIP := cidr.ToAddressRange()
	return ourFirstIP.Compare(lastIP) > 0
}

func (in *CIDR) Join(cidr *CIDR) {
	if !in.CanJoin(cidr) {
		return
	}
	ourOnes := in.Net.Bits()
	ourBits := in.Net.Addr().BitLen()

	joinOnes := ourOnes - 1
	joinBitsDiff := ourBits - joinOnes
	joinIPBitGlobalIdx := joinBitsDiff - 1
	joinIPBitLocalIdx := joinBitsDiff % 8
	if joinIPBitLocalIdx == 0 {
		joinIPBitLocalIdx = 8
	}

	var ipBytes []byte
	firstOtherIP, _ := in.ToAddressRange()

	if in.IsIPv4() {
		ipv4 := firstOtherIP.As4()
		ipBytes = ipv4[:]
	} else {
		ipv6 := firstOtherIP.As16()
		ipBytes = ipv6[:]
	}
	ipLen := len(ipBytes)

	joinIPIdx := ipLen - joinIPBitGlobalIdx/8 - 1
	ipBytes[joinIPIdx] &= 0xff << joinIPBitLocalIdx

	if in.IsIPv6() {
		ipv6 := (*[16]byte)(ipBytes)
		ip := netip.PrefixFrom(netip.AddrFrom16(*ipv6), joinOnes)
		in.Net = ip
		return
	}
	ipv4 := (*[4]byte)(ipBytes)
	ip := netip.PrefixFrom(netip.AddrFrom4(*ipv4), joinOnes)
	in.Net = ip
}

func (in *CIDR) CanJoin(cidr *CIDR) bool {
	ourOnes := in.Net.Bits()
	ourBits := in.Net.Addr().BitLen()

	theirOnes := cidr.Net.Bits()
	theirBits := cidr.Net.Addr().BitLen()

	ourBitsDiff := ourBits - ourOnes
	if ourBitsDiff == ourBits {
		return false
	}
	otherBitsDiff := ourBitsDiff

	var otherIPAddr netip.Addr
	var otherIP []byte
	firstOtherIP, _ := in.ToAddressRange()
	firstTheirIP, _ := cidr.ToAddressRange()

	isIPv6 := in.IsIPv6()

	if isIPv6 {
		ipv6 := firstOtherIP.As16()
		otherIP = ipv6[:]
	} else {
		ipv4 := firstOtherIP.As4()
		otherIP = ipv4[:]
	}

	ipLen := len(otherIP)
	otherIPIdx := ipLen - otherBitsDiff/8 - 1
	otherIPBit := otherBitsDiff % 8
	otherIP[otherIPIdx] = otherIP[otherIPIdx] ^ (1 << otherIPBit)

	if isIPv6 {
		ipv6 := (*[16]byte)(otherIP)
		otherIPAddr = netip.AddrFrom16(*ipv6)
	} else {
		ipv4 := (*[4]byte)(otherIP)
		otherIPAddr = netip.AddrFrom4(*ipv4)
	}

	if firstTheirIP.Compare(otherIPAddr) == 0 &&
		theirOnes == ourOnes &&
		theirBits == ourBits {
		return true
	}
	return false
}

func (in *CIDR) Reserve(cidr *CIDR) []CIDR {
	ourOnes := in.Net.Bits()
	ourBits := in.Net.Addr().BitLen()
	theirOnes := cidr.Net.Bits()
	theirBits := cidr.Net.Addr().BitLen()

	// Check if addresses/masks are the same length
	if ourBits != theirBits {
		return []CIDR{*in}
	}

	// Check if their mask capacity is bigger then ours
	if ourOnes > theirOnes {
		return []CIDR{*in}
	}

	// If capacities are equal, then net IPs should be also equal
	// Otherwise networks are not the same
	if ourOnes == theirOnes && in.Net.Addr().Compare(cidr.Net.Addr()) == 0 {
		return []CIDR{}
	}

	onesDiff := theirOnes - ourOnes
	nets := make([]CIDR, onesDiff)
	leftInsertIdx := 0
	rightInsertIdx := int(onesDiff) - 1
	splitOnes := ourOnes + 1

	isIPv6 := in.IsIPv6()
	var currentIP []byte
	theirFirstIP, _ := cidr.ToAddressRange()
	if isIPv6 {
		ipv6 := in.Net.Addr().As16()
		currentIP = ipv6[:]
	} else {
		firstIP, _ := in.ToAddressRange()
		ipv4 := firstIP.As4()
		currentIP = ipv4[:]
	}
	ipLen := len(currentIP)

	for leftInsertIdx <= rightInsertIdx {
		leftIP := make([]byte, ipLen)
		copy(leftIP, currentIP)
		rightIP := make([]byte, ipLen)
		copy(rightIP, currentIP)

		prevBitsDiff := ourBits - splitOnes
		rightIPIdx := ipLen - prevBitsDiff/8 - 1
		rightIPBit := prevBitsDiff % 8
		rightIP[rightIPIdx] = rightIP[rightIPIdx] | (1 << rightIPBit)

		var leftNet, rightNet netip.Prefix
		if isIPv6 {
			leftIpv6Bytes := (*[16]byte)(leftIP)
			rightIpv6Bytes := (*[16]byte)(rightIP)
			leftNet = netip.PrefixFrom(netip.AddrFrom16(*leftIpv6Bytes), splitOnes)
			rightNet = netip.PrefixFrom(netip.AddrFrom16(*rightIpv6Bytes), splitOnes)
		} else {
			leftIpv4Bytes := (*[4]byte)(leftIP)
			rightIpv4Bytes := (*[4]byte)(rightIP)
			leftNet = netip.PrefixFrom(netip.AddrFrom4(*leftIpv4Bytes), splitOnes)
			rightNet = netip.PrefixFrom(netip.AddrFrom4(*rightIpv4Bytes), splitOnes)
		}

		if leftNet.Contains(theirFirstIP) {
			nets[rightInsertIdx] = *CIDRFromNet(rightNet)
			rightInsertIdx = rightInsertIdx - 1
			currentIP = leftIP
		} else {
			nets[leftInsertIdx] = *CIDRFromNet(leftNet)
			leftInsertIdx = leftInsertIdx + 1
			currentIP = rightIP
		}

		splitOnes = splitOnes + 1
	}

	return nets
}

func (in *CIDR) CanReserve(cidr *CIDR) bool {
	ourOnes := in.Net.Bits()
	ourBits := in.Net.Addr().BitLen()
	theirOnes := cidr.Net.Bits()
	theirBits := cidr.Net.Addr().BitLen()

	// Check if addresses/masks are the same length
	if ourBits != theirBits {
		return false
	}

	// Check if their mask capacity is bigger then ours
	if ourOnes > theirOnes {
		return false
	}

	ourFirstIP, _ := in.ToAddressRange()
	theirFirstIP, _ := cidr.ToAddressRange()

	// If capacities are equal, then net IPs should be also equal
	// Otherwise networks are not the same
	if ourOnes == theirOnes {
		return ourFirstIP.Compare(theirFirstIP) == 0
	}

	if !in.Net.Contains(cidr.Net.Addr()) {
		return false
	}
	return true
}

func (in *CIDR) IsIPv4() bool {
	return in.Net.Addr().Is4()
}

func (in *CIDR) IsIPv6() bool {
	return in.Net.Addr().Is6()
}

func (in *CIDR) String() string {
	return in.Net.String()
}

func (in *CIDR) AsIPAddr() *IPAddr {
	return &IPAddr{
		Net: in.Net.Addr(),
	}
}

// DeepCopyInto is an deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CIDR) DeepCopyInto(out *CIDR) {
	*out = *in
	if in.Net.Addr().String() != "" {
		out.Net = in.Net
	}
}
