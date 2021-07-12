package v1alpha1

import (
	"math/big"
	"net"

	"k8s.io/apimachinery/pkg/util/json"
)

// +kubebuilder:validation:Type=string
type CIDR struct {
	Net *net.IPNet `json:"-"`
}

func CIDRFromString(cidrString string) (*CIDR, error) {
	_, cidr, err := net.ParseCIDR(cidrString)
	if err != nil {
		return nil, err
	}
	return &CIDR{
		Net: cidr,
	}, nil
}

func CIDRFromNet(n *net.IPNet) *CIDR {
	return &CIDR{
		Net: n,
	}
}

func (in CIDR) MarshalJSON() ([]byte, error) {
	cidr := in.String()
	return json.Marshal(cidr)
}

func (in *CIDR) UnmarshalJSON(b []byte) error {
	stringVal := string(b)
	if stringVal == "null" {
		return nil
	}
	if err := json.Unmarshal(b, &stringVal); err != nil {
		return err
	}
	_, nw, err := net.ParseCIDR(stringVal)
	if err != nil {
		return err
	}
	in.Net = nw

	return nil
}

func (in *CIDR) MaskBits() byte {
	_, bits := in.Net.Mask.Size()

	return byte(bits)
}

func (in *CIDR) MaskOnes() byte {
	ones, _ := in.Net.Mask.Size()

	return byte(ones)
}

func (in *CIDR) MaskZeroes() byte {
	ones, bits := in.Net.Mask.Size()

	return byte(bits - ones)
}

func (in *CIDR) AddressCapacity() *big.Int {
	ones, bits := in.Net.Mask.Size()

	ac := big.Int{}
	ac.Exp(big.NewInt(2), big.NewInt(int64(bits-ones)), nil)

	return &ac
}

func (in *CIDR) MaskCapacity() *big.Int {
	count := in.AddressCapacity()
	count.Sub(count, big.NewInt(1))

	return count
}

func (in *CIDR) ToAddressRange() (net.IP, net.IP) {
	firstIp := in.IPBytes()

	first := big.Int{}
	first.SetBytes(firstIp)

	last := big.Int{}
	last.Add(&first, in.MaskCapacity())
	lastIP := make(net.IP, len(firstIp))
	last.FillBytes(lastIP)

	return firstIp, lastIP
}

func (in *CIDR) Equal(cidr *CIDR) bool {
	ourOnes, ourBits := in.Net.Mask.Size()
	theirOnes, theirBits := cidr.Net.Mask.Size()

	return in.IPBytes().Equal(cidr.IPBytes()) &&
		ourBits == theirBits &&
		ourOnes == theirOnes
}

func (in *CIDR) isLeft(ones int, bits int) bool {
	ipBytes := in.IPBytes()
	ipLen := len(ipBytes)
	bitsDiff := bits - ones
	ipIdx := ipLen - bitsDiff/8 - 1
	ipBit := bitsDiff % 8
	return ipBytes[ipIdx]&(1<<ipBit) == 0
}

func (in *CIDR) IsLeft() bool {
	ones, bits := in.Net.Mask.Size()
	if ones == 0 {
		return false
	}
	return in.isLeft(ones, bits)
}

func (in *CIDR) IsRight() bool {
	ones, bits := in.Net.Mask.Size()
	if ones == 0 {
		return false
	}
	return !in.isLeft(ones, bits)
}

func (in *CIDR) Before(cidr *CIDR) bool {
	_, lastIP := in.ToAddressRange()
	ourLast := big.Int{}
	ourLast.SetBytes(lastIP)

	theirFirst := big.Int{}
	theirFirst.SetBytes(cidr.IPBytes())

	return ourLast.Cmp(&theirFirst) < 0
}

func (in *CIDR) After(cidr *CIDR) bool {
	ourFirst := big.Int{}
	ourFirst.SetBytes(in.IPBytes())

	_, lastIP := cidr.ToAddressRange()
	theirLast := big.Int{}
	theirLast.SetBytes(lastIP)

	return ourFirst.Cmp(&theirLast) > 0
}

func (in *CIDR) Join(cidr *CIDR) {
	if !in.CanJoin(cidr) {
		return
	}

	ipBytes := in.IPBytes()
	ipLen := len(ipBytes)
	ourOnes, ourBits := in.Net.Mask.Size()
	joinOnes := ourOnes - 1
	joinBitsDiff := ourBits - joinOnes
	joinIPBitGlobalIdx := joinBitsDiff - 1
	joinIPBitLocalIdx := joinBitsDiff % 8
	if joinIPBitLocalIdx == 0 {
		joinIPBitLocalIdx = 8
	}
	joinIPIdx := ipLen - joinIPBitGlobalIdx/8 - 1
	ipBytes[joinIPIdx] = ipBytes[joinIPIdx] & (0xff << joinIPBitLocalIdx)
	in.Net.IP = ipBytes
	in.Net.Mask = net.CIDRMask(joinOnes, ourBits)
}

func (in *CIDR) CanJoin(cidr *CIDR) bool {
	ourOnes, ourBits := in.Net.Mask.Size()
	ourBitsDiff := ourBits - ourOnes
	if ourBitsDiff == ourBits {
		return false
	}

	ourIp := in.IPBytes()
	ipLen := len(ourIp)
	otherIP := make(net.IP, ipLen)
	copy(otherIP, ourIp)

	otherBitsDiff := ourBitsDiff
	otherIPIdx := ipLen - otherBitsDiff/8 - 1
	otherIPBit := otherBitsDiff % 8
	otherIP[otherIPIdx] = otherIP[otherIPIdx] ^ (1 << otherIPBit)

	theirIP := cidr.IPBytes()
	theirOnes, theirBits := cidr.Net.Mask.Size()
	if theirIP.Equal(otherIP) &&
		theirOnes == ourOnes &&
		theirBits == ourBits {
		return true
	}

	return false
}

func (in *CIDR) Reserve(cidr *CIDR) []CIDR {
	ourOnes, ourBits := in.Net.Mask.Size()
	theirOnes, theirBits := cidr.Net.Mask.Size()

	// Check if addresses/masks are the same length
	if ourBits != theirBits {
		return []CIDR{*in}
	}

	// Check if their mask capacity is bigger then ours
	if ourOnes > theirOnes {
		return []CIDR{*in}
	}

	ourIp := in.IPBytes()
	theirIp := cidr.IPBytes()
	// If capacities are equal, then net IPs should be also equal
	// Otherwise networks are not the same
	if ourOnes == theirOnes && ourIp.Equal(theirIp) {
		return []CIDR{}
	}

	onesDiff := theirOnes - ourOnes
	nets := make([]CIDR, onesDiff)
	leftInsertIdx := 0
	rightInsertIdx := onesDiff - 1
	splitOnes := ourOnes + 1
	currentIP := ourIp
	ipLen := len(currentIP)
	for leftInsertIdx <= rightInsertIdx {
		leftIP := make(net.IP, ipLen)
		copy(leftIP, currentIP)

		rightIP := make(net.IP, ipLen)
		copy(rightIP, currentIP)
		prevBitsDiff := ourBits - splitOnes
		rightIPIdx := ipLen - prevBitsDiff/8 - 1
		rightIPBit := prevBitsDiff % 8
		rightIP[rightIPIdx] = rightIP[rightIPIdx] | (1 << rightIPBit)

		mask := net.CIDRMask(splitOnes, ourBits)

		leftNet := &net.IPNet{
			IP:   leftIP,
			Mask: mask,
		}

		rightNet := &net.IPNet{
			IP:   rightIP,
			Mask: mask,
		}

		if leftNet.Contains(theirIp) {
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
	ourOnes, ourBits := in.Net.Mask.Size()
	theirOnes, theirBits := cidr.Net.Mask.Size()

	// Check if addresses/masks are the same length
	if ourBits != theirBits {
		return false
	}

	// Check if their mask capacity is bigger then ours
	if ourOnes > theirOnes {
		return false
	}

	ourIp := in.IPBytes()
	theirIp := cidr.IPBytes()
	// If capacities are equal, then net IPs should be also equal
	// Otherwise networks are not the same
	if ourOnes == theirOnes {
		return ourIp.Equal(theirIp)
	}

	for i := range in.Net.IP {
		if ourIp[i]&in.Net.Mask[i] != theirIp[i]&cidr.Net.Mask[i]&in.Net.Mask[i] {
			return false
		}
	}

	return true
}

func (in *CIDR) IsIPv4() bool {
	return in.Net.IP.To4() != nil
}

func (in *CIDR) IsIPv6() bool {
	return !in.IsIPv4()
}

func (in *CIDR) String() string {
	return in.Net.String()
}

func (in *CIDR) IPBytes() net.IP {
	ip := in.Net.IP.To4()
	if ip == nil {
		ip = in.Net.IP.To16()
	}
	return ip
}

// DeepCopyInto is an deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CIDR) DeepCopyInto(out *CIDR) {
	*out = *in
	if in.Net != nil {
		ip := make(net.IP, len(in.Net.IP))
		copy(ip, in.Net.IP)
		mask := make(net.IPMask, len(in.Net.Mask))
		copy(mask, in.Net.Mask)
		nw := net.IPNet{
			IP:   ip,
			Mask: mask,
		}
		out.Net = &nw
	}
	return
}
