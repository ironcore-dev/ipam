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

	"inet.af/netaddr"
)

// +kubebuilder:validation:Type=string
type CIDR struct {
	Net netaddr.IPPrefix `json:"-"`
}

func CIDRFromString(cidrString string) (*CIDR, error) {
	cidr, err := netaddr.ParseIPPrefix(cidrString)
	if err != nil {
		return nil, err
	}
	return &CIDR{
		Net: cidr,
	}, nil
}

func CIDRFromNet(n netaddr.IPPrefix) *CIDR {
	return &CIDR{Net: n}
}

func (in CIDR) MarshalJSON() ([]byte, error) {
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
	pIP, err := netaddr.ParseIPPrefix(stringVal)
	if err != nil {
		return err
	}
	in.Net = pIP
	return nil
}

func (in *CIDR) MaskBits() byte {
	return in.Net.IP().BitLen()
}

func (in *CIDR) MaskOnes() byte {
	return in.Net.Bits()
}

func (in *CIDR) MaskZeroes() byte {
	ones := in.Net.Bits()
	bits := in.Net.IP().BitLen()
	return bits - ones
}

func (in *CIDR) AddressCapacity() *big.Int {
	ones := in.Net.Bits()
	bits := in.Net.IP().BitLen()

	ac := big.Int{}
	ac.Exp(big.NewInt(2), big.NewInt(int64(bits-ones)), nil)

	return &ac
}

func (in *CIDR) MaskCapacity() *big.Int {
	count := in.AddressCapacity()
	count.Sub(count, big.NewInt(1))

	return count
}

func (in *CIDR) ToAddressRange() (netaddr.IP, netaddr.IP) {
	return in.Net.Range().From(), in.Net.Range().To()
}

func (in *CIDR) Equal(cidr *CIDR) bool {
	ourOnes := in.Net.Bits()
	ourBits := in.Net.IP().BitLen()

	theirOnes := cidr.Net.Bits()
	theirBits := cidr.Net.IP().BitLen()
	firstOurIP, _ := in.ToAddressRange()
	firstTheirIP, _ := cidr.ToAddressRange()

	return firstOurIP.Compare(firstTheirIP) == 0 &&
		ourBits == theirBits &&
		ourOnes == theirOnes
}

func (in *CIDR) IsLeft() bool {
	ones := in.Net.Bits()
	bits := in.Net.IP().BitLen()
	if ones == 0 {
		return false
	}
	return in.isLeft(ones, bits)
}

func (in *CIDR) isLeft(ones, bits uint8) bool {
	var ipBytes []byte
	if in.Net.IP().Is4() {
		ipv4 := in.Net.IP().As4()
		ipBytes = ipv4[:]
	} else {
		ipv6 := in.Net.IP().As16()
		ipBytes = ipv6[:]
	}
	ipLen := len(ipBytes)
	bitsDiff := bits - ones
	ipIdx := uint8(ipLen) - bitsDiff/8 - 1
	ipBit := bitsDiff % 8
	return ipBytes[ipIdx]&(1<<ipBit) == 0
}

func (in *CIDR) IsRight() bool {
	ones := in.Net.Bits()
	bits := in.Net.IP().BitLen()
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
	ourBits := in.Net.IP().BitLen()

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

	joinIPIdx := uint8(ipLen) - joinIPBitGlobalIdx/8 - 1
	ipBytes[joinIPIdx] &= 0xff << joinIPBitLocalIdx

	if in.IsIPv6() {
		ipv6 := (*[16]byte)(ipBytes)
		ip := netaddr.IPPrefixFrom(netaddr.IPFrom16(*ipv6), joinOnes)
		in.Net = ip
		return
	}
	ipv4 := (*[4]byte)(ipBytes)
	ip := netaddr.IPPrefixFrom(netaddr.IPFrom4(*ipv4), joinOnes)
	in.Net = ip
}

func (in *CIDR) CanJoin(cidr *CIDR) bool {
	ourOnes := in.Net.Bits()
	ourBits := in.Net.IP().BitLen()

	theirOnes := cidr.Net.Bits()
	theirBits := cidr.Net.IP().BitLen()

	ourBitsDiff := ourBits - ourOnes
	if ourBitsDiff == ourBits {
		return false
	}
	otherBitsDiff := ourBitsDiff

	var otherIPAddr netaddr.IP
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

	ipLen := uint8(len(otherIP))
	otherIPIdx := ipLen - otherBitsDiff/8 - 1
	otherIPBit := otherBitsDiff % 8
	otherIP[otherIPIdx] = otherIP[otherIPIdx] ^ (1 << otherIPBit)

	if isIPv6 {
		ipv6 := (*[16]byte)(otherIP)
		otherIPAddr = netaddr.IPFrom16(*ipv6)
	} else {
		ipv4 := (*[4]byte)(otherIP)
		otherIPAddr = netaddr.IPFrom4(*ipv4)
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
	ourBits := in.Net.IP().BitLen()
	theirOnes := cidr.Net.Bits()
	theirBits := cidr.Net.IP().BitLen()

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
	if ourOnes == theirOnes && in.Net.IP().Compare(cidr.Net.IP()) == 0 {
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
		ipv6 := in.Net.IP().As16()
		currentIP = ipv6[:]
	} else {
		firstIP, _ := in.ToAddressRange()
		ipv4 := firstIP.As4()
		currentIP = ipv4[:]
	}
	ipLen := uint8(len(currentIP))

	for leftInsertIdx <= rightInsertIdx {
		leftIP := make([]byte, ipLen)
		copy(leftIP, currentIP)
		rightIP := make([]byte, ipLen)
		copy(rightIP, currentIP)

		prevBitsDiff := ourBits - splitOnes
		rightIPIdx := ipLen - prevBitsDiff/8 - 1
		rightIPBit := prevBitsDiff % 8
		rightIP[rightIPIdx] = rightIP[rightIPIdx] | (1 << rightIPBit)

		var leftNet, rightNet netaddr.IPPrefix
		if isIPv6 {
			leftIpv6Bytes := (*[16]byte)(leftIP)
			rightIpv6Bytes := (*[16]byte)(rightIP)
			leftNet = netaddr.IPPrefixFrom(netaddr.IPFrom16(*leftIpv6Bytes), splitOnes)
			rightNet = netaddr.IPPrefixFrom(netaddr.IPFrom16(*rightIpv6Bytes), splitOnes)
		} else {
			leftIpv4Bytes := (*[4]byte)(leftIP)
			rightIpv4Bytes := (*[4]byte)(rightIP)
			leftNet = netaddr.IPPrefixFrom(netaddr.IPFrom4(*leftIpv4Bytes), splitOnes)
			rightNet = netaddr.IPPrefixFrom(netaddr.IPFrom4(*rightIpv4Bytes), splitOnes)
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
	ourBits := in.Net.IP().BitLen()
	theirOnes := cidr.Net.Bits()
	theirBits := cidr.Net.IP().BitLen()

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

	if !in.Net.Contains(cidr.Net.IP()) {
		return false
	}
	return true
}

func (in *CIDR) IsIPv4() bool {
	return in.Net.IP().Is4()
}

func (in *CIDR) IsIPv6() bool {
	return in.Net.IP().Is6()
}

func (in *CIDR) String() string {
	return in.Net.String()
}

func (in *CIDR) AsIPAddr() *IPAddr {
	return &IPAddr{
		Net: in.Net.IP(),
	}
}

// DeepCopyInto is an deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CIDR) DeepCopyInto(out *CIDR) {
	*out = *in
	if in.Net.IP().String() != "" {
		out.Net = in.Net
	}
}
