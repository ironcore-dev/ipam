package v1alpha1

import (
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
	return in.Net.MarshalText()
}

func (in *CIDR) UnmarshalJSON(b []byte) error {
	return in.Net.UnmarshalText(b)
}

func (in *CIDR) MaskBits() byte {
	return byte(in.Net.IP().BitLen())
}

func (in *CIDR) MaskOnes() byte {
	return byte(in.Net.Bits())
}

func (in *CIDR) MaskZeroes() byte {
	ones := in.Net.Bits()
	bits := in.Net.IP().BitLen()
	return byte(bits - ones)
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

	theirOnes := in.Net.Bits()
	theirBits := in.Net.IP().BitLen()

	return in.Net.IP().Compare(cidr.Net.IP()) == 0 &&
		ourBits == theirBits &&
		ourOnes == theirOnes
}

func (in *CIDR) isLeft(ones, bits uint8) bool {
	ipBytes := in.IPBytes()
	ipLen := len(ipBytes)
	bitsDiff := bits - ones
	ipIdx := uint8(ipLen) - bitsDiff/8 - 1
	ipBit := bitsDiff % 8
	return ipBytes[ipIdx]&(1<<ipBit) == 0
}

func (in *CIDR) IsLeft() bool {
	ones := in.Net.Bits()
	bits := in.Net.IP().BitLen()
	if ones == 0 {
		return false
	}
	return in.isLeft(ones, bits)
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
	firstIP, _ := in.ToAddressRange()
	return firstIP.Compare(cidr.Net.IP()) > 0
}

func (in *CIDR) After(cidr *CIDR) bool {
	_, lastIP := in.ToAddressRange()
	return lastIP.Compare(cidr.Net.IP()) < 0
}

func (in *CIDR) Join(cidr *CIDR) {
	if !in.CanJoin(cidr) {
		return
	}

	ipBytes := in.IPBytes()
	ipLen := len(ipBytes)

	ourOnes := in.Net.Bits()
	ourBits := in.Net.IP().BitLen()

	joinOnes := ourOnes - 1
	joinBitsDiff := ourBits - joinOnes
	joinIPBitGlobalIdx := joinBitsDiff - 1
	joinIPBitLocalIdx := joinBitsDiff % 8
	if joinIPBitLocalIdx == 0 {
		joinIPBitLocalIdx = 8
	}
	joinIPIdx := uint8(ipLen) - joinIPBitGlobalIdx/8 - 1
	ipBytes[joinIPIdx] = ipBytes[joinIPIdx] & (0xff << joinIPBitLocalIdx)

	if err := in.Net.UnmarshalText(ipBytes); err != nil {
		return
	}
}

func (in *CIDR) CanJoin(cidr *CIDR) bool {
	ourOnes := in.Net.Bits()
	ourBits := in.Net.IP().BitLen()

	ourBitsDiff := ourBits - ourOnes
	if ourBitsDiff == ourBits {
		return false
	}

	switch {
	case in.Net.IP().Is4():
		ourIp := in.Net.IP().As4()
		ipLen := uint8(len(ourIp))
		var otherIP [4]byte
		otherIP = ourIp

		otherBitsDiff := ourBitsDiff
		otherIPIdx := ipLen - otherBitsDiff/8 - 1
		otherIPBit := otherBitsDiff % 8
		otherIP[otherIPIdx] = otherIP[otherIPIdx] ^ (1 << otherIPBit)

		theirOnes := in.Net.Bits()
		theirBits := in.Net.IP().BitLen()

		if cidr.Net.IP().Compare(netaddr.IPFrom4(otherIP)) == 0 &&
			theirOnes == ourOnes &&
			theirBits == ourBits {
			return true
		}
	case in.Net.IP().Is6():
		ourIp := in.Net.IP().As16()
		ipLen := uint8(len(ourIp))
		var otherIP [16]byte
		otherIP = ourIp

		otherBitsDiff := ourBitsDiff
		otherIPIdx := ipLen - otherBitsDiff/8 - 1
		otherIPBit := otherBitsDiff % 8
		otherIP[otherIPIdx] = otherIP[otherIPIdx] ^ (1 << otherIPBit)

		theirOnes := in.Net.Bits()
		theirBits := in.Net.IP().BitLen()

		if cidr.Net.IP().Compare(netaddr.IPFrom16(otherIP)) == 0 &&
			theirOnes == ourOnes &&
			theirBits == ourBits {
			return true
		}
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
	switch {
	case in.IsIPv4():
		currentIP := in.Net.Masked().IP().As4()
		ipLen := uint8(len(currentIP))
		for leftInsertIdx <= rightInsertIdx {
			leftIP := currentIP
			rightIP := currentIP

			prevBitsDiff := ourBits - splitOnes
			rightIPIdx := ipLen - prevBitsDiff/8 - 1
			rightIPBit := prevBitsDiff % 8
			rightIP[rightIPIdx] = rightIP[rightIPIdx] | (1 << rightIPBit)

			leftNet := netaddr.IPPrefixFrom(netaddr.IPFrom4(leftIP), splitOnes)
			rightNet := netaddr.IPPrefixFrom(netaddr.IPFrom4(rightIP), splitOnes)

			if leftNet.Contains(cidr.Net.IP()) {
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
	case in.IsIPv6():
		currentIP := in.Net.Masked().IP().As16()
		ipLen := uint8(len(currentIP))
		for leftInsertIdx <= rightInsertIdx {
			leftIP := currentIP
			rightIP := currentIP

			prevBitsDiff := ourBits - splitOnes
			rightIPIdx := ipLen - prevBitsDiff/8 - 1
			rightIPBit := prevBitsDiff % 8
			rightIP[rightIPIdx] = rightIP[rightIPIdx] | (1 << rightIPBit)

			leftNet := netaddr.IPPrefixFrom(netaddr.IPFrom16(leftIP), splitOnes)
			rightNet := netaddr.IPPrefixFrom(netaddr.IPFrom16(rightIP), splitOnes)

			if leftNet.Contains(cidr.Net.IP()) {
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

	if ourOnes == theirOnes {
		return in.Net.Contains(cidr.Net.IP())
	}

	// If capacities are equal, then net IPs should be also equal
	// Otherwise networks are not the same
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

func (in *CIDR) IPBytes() []byte {
	return []byte(in.Net.IP().String())
}

// DeepCopyInto is an deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CIDR) DeepCopyInto(out *CIDR) {
	if !in.Net.IP().IsZero() {
		out.Net = in.Net
	}
}
