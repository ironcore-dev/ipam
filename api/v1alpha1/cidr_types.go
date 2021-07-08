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

func (n CIDR) MarshalJSON() ([]byte, error) {
	cidr := n.String()
	return json.Marshal(cidr)
}

func (n *CIDR) UnmarshalJSON(b []byte) error {
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
	n.Net = nw

	return nil
}

func (n *CIDR) MaskBits() byte {
	_, bits := n.Net.Mask.Size()

	return byte(bits)
}

func (n *CIDR) MaskOnes() byte {
	ones, _ := n.Net.Mask.Size()

	return byte(ones)
}

func (n *CIDR) MaskZeroes() byte {
	ones, bits := n.Net.Mask.Size()

	return byte(bits - ones)
}

func (n *CIDR) AddressCapacity() *big.Int {
	ones, bits := n.Net.Mask.Size()

	ac := big.Int{}
	ac.Exp(big.NewInt(2), big.NewInt(int64(bits-ones)), nil)

	return &ac
}

func (n *CIDR) MaskCapacity() *big.Int {
	count := n.AddressCapacity()
	count.Sub(count, big.NewInt(1))

	return count
}

func (n *CIDR) ToAddressRange() (net.IP, net.IP) {
	firstIp := n.IPBytes()

	first := big.Int{}
	first.SetBytes(firstIp)

	last := big.Int{}
	last.Add(&first, n.MaskCapacity())
	lastIP := make(net.IP, len(firstIp))
	last.FillBytes(lastIP)

	return firstIp, lastIP
}

func (n *CIDR) Equal(cidr *CIDR) bool {
	ourOnes, ourBits := n.Net.Mask.Size()
	theirOnes, theirBits := cidr.Net.Mask.Size()

	return n.IPBytes().Equal(cidr.IPBytes()) &&
		ourBits == theirBits &&
		ourOnes == theirOnes
}

func (n *CIDR) isLeft(ones int, bits int) bool {
	ipBytes := n.IPBytes()
	ipLen := len(ipBytes)
	bitsDiff := bits - ones
	ipIdx := ipLen - bitsDiff/8 - 1
	ipBit := bitsDiff % 8
	return ipBytes[ipIdx]&(1<<ipBit) == 0
}

func (n *CIDR) IsLeft() bool {
	ones, bits := n.Net.Mask.Size()
	if ones == 0 {
		return false
	}
	return n.isLeft(ones, bits)
}

func (n *CIDR) IsRight() bool {
	ones, bits := n.Net.Mask.Size()
	if ones == 0 {
		return false
	}
	return !n.isLeft(ones, bits)
}

func (n *CIDR) Before(cidr *CIDR) bool {
	_, lastIP := n.ToAddressRange()
	ourLast := big.Int{}
	ourLast.SetBytes(lastIP)

	theirFirst := big.Int{}
	theirFirst.SetBytes(cidr.IPBytes())

	return ourLast.Cmp(&theirFirst) < 0
}

func (n *CIDR) After(cidr *CIDR) bool {
	ourFirst := big.Int{}
	ourFirst.SetBytes(n.IPBytes())

	_, lastIP := cidr.ToAddressRange()
	theirLast := big.Int{}
	theirLast.SetBytes(lastIP)

	return ourFirst.Cmp(&theirLast) > 0
}

func (n *CIDR) Join(cidr *CIDR) {
	if !n.CanJoin(cidr) {
		return
	}

	ipBytes := n.IPBytes()
	ipLen := len(ipBytes)
	ourOnes, ourBits := n.Net.Mask.Size()
	joinOnes := ourOnes - 1
	joinBitsDiff := ourBits - joinOnes
	joinIPBitGlobalIdx := joinBitsDiff - 1
	joinIPBitLocalIdx := joinBitsDiff % 8
	if joinIPBitLocalIdx == 0 {
		joinIPBitLocalIdx = 8
	}
	joinIPIdx := ipLen - joinIPBitGlobalIdx/8 - 1
	ipBytes[joinIPIdx] = ipBytes[joinIPIdx] & (0xff << joinIPBitLocalIdx)
	n.Net.IP = ipBytes
	n.Net.Mask = net.CIDRMask(joinOnes, ourBits)
}

func (n *CIDR) CanJoin(cidr *CIDR) bool {
	ourOnes, ourBits := n.Net.Mask.Size()
	ourBitsDiff := ourBits - ourOnes
	if ourBitsDiff == ourBits {
		return false
	}

	ourIp := n.IPBytes()
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

func (n *CIDR) Reserve(cidr *CIDR) []CIDR {
	ourOnes, ourBits := n.Net.Mask.Size()
	theirOnes, theirBits := cidr.Net.Mask.Size()

	// Check if addresses/masks are the same length
	if ourBits != theirBits {
		return []CIDR{*n}
	}

	// Check if their mask capacity is bigger then ours
	if ourOnes > theirOnes {
		return []CIDR{*n}
	}

	ourIp := n.IPBytes()
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

func (n *CIDR) CanReserve(cidr *CIDR) bool {
	ourOnes, ourBits := n.Net.Mask.Size()
	theirOnes, theirBits := cidr.Net.Mask.Size()

	// Check if addresses/masks are the same length
	if ourBits != theirBits {
		return false
	}

	// Check if their mask capacity is bigger then ours
	if ourOnes > theirOnes {
		return false
	}

	ourIp := n.IPBytes()
	theirIp := cidr.IPBytes()
	// If capacities are equal, then net IPs should be also equal
	// Otherwise networks are not the same
	if ourOnes == theirOnes {
		return ourIp.Equal(theirIp)
	}

	for i := range n.Net.IP {
		if ourIp[i]&n.Net.Mask[i] != theirIp[i]&cidr.Net.Mask[i]&n.Net.Mask[i] {
			return false
		}
	}

	return true
}

func (n *CIDR) IsIPv4() bool {
	return n.Net.IP.To4() != nil
}

func (n *CIDR) IsIPv6() bool {
	return !n.IsIPv4()
}

func (n *CIDR) String() string {
	return n.Net.String()
}

func (n *CIDR) IPBytes() net.IP {
	ip := n.Net.IP.To4()
	if ip == nil {
		ip = n.Net.IP.To16()
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
