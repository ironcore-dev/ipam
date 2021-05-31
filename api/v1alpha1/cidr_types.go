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
	first := big.Int{}
	first.SetBytes(n.Net.IP)

	last := big.Int{}
	last.Add(&first, n.MaskCapacity())
	lastIP := make(net.IP, len(n.Net.IP))
	last.FillBytes(lastIP)

	return n.Net.IP, lastIP
}

func (n *CIDR) Equal(cidr *CIDR) bool {
	ourOnes, ourBits := n.Net.Mask.Size()
	theirOnes, theirBits := cidr.Net.Mask.Size()

	return n.Net.IP.Equal(cidr.Net.IP) &&
		ourBits == theirBits &&
		ourOnes == theirOnes
}

func (n *CIDR) isLeft(ones int, bits int) bool {
	ipLen := len(n.Net.IP)
	bitsDiff := bits - ones
	ipIdx := ipLen - bitsDiff/8 - 1
	ipBit := bitsDiff % 8
	return n.Net.IP[ipIdx]&(1<<ipBit) == 0
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
	theirFirst.SetBytes(cidr.Net.IP)

	return ourLast.Cmp(&theirFirst) < 0
}

func (n *CIDR) After(cidr *CIDR) bool {
	ourFirst := big.Int{}
	ourFirst.SetBytes(n.Net.IP)

	_, lastIP := cidr.ToAddressRange()
	theirLast := big.Int{}
	theirLast.SetBytes(lastIP)

	return ourFirst.Cmp(&theirLast) > 0
}

func (n *CIDR) Join(cidr *CIDR) {
	if !n.CanJoin(cidr) {
		return
	}

	ipLen := len(n.Net.IP)
	ourOnes, ourBits := n.Net.Mask.Size()
	joinOnes := ourOnes - 1
	joinBitsDiff := ourBits - joinOnes
	joinIPBitGlobalIdx := joinBitsDiff - 1
	joinIPBitLocalIdx := joinBitsDiff % 8
	if joinIPBitLocalIdx == 0 {
		joinIPBitLocalIdx = 8
	}
	joinIPIdx := ipLen - joinIPBitGlobalIdx/8 - 1
	n.Net.IP[joinIPIdx] = n.Net.IP[joinIPIdx] & (0xff << joinIPBitLocalIdx)
	n.Net.Mask = net.CIDRMask(joinOnes, ourBits)
}

func (n *CIDR) CanJoin(cidr *CIDR) bool {
	ourOnes, ourBits := n.Net.Mask.Size()
	ourBitsDiff := ourBits - ourOnes
	if ourBitsDiff == ourBits {
		return false
	}

	ipLen := len(n.Net.IP)
	otherIP := make(net.IP, len(n.Net.IP))
	copy(otherIP, n.Net.IP)

	otherBitsDiff := ourBitsDiff
	otherIPIdx := ipLen - otherBitsDiff/8 - 1
	otherIPBit := otherBitsDiff % 8
	otherIP[otherIPIdx] = otherIP[otherIPIdx] ^ (1 << otherIPBit)

	theirOnes, theirBits := cidr.Net.Mask.Size()
	if cidr.Net.IP.Equal(otherIP) &&
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

	// If capacities are equal, then net IPs should be also equal
	// Otherwise networks are not the same
	if ourOnes == theirOnes && n.Net.IP.Equal(n.Net.IP) {
		return []CIDR{}
	}

	onesDiff := theirOnes - ourOnes
	nets := make([]CIDR, onesDiff)
	leftInsertIdx := 0
	rightInsertIdx := onesDiff - 1
	splitOnes := ourOnes + 1
	currentIP := n.Net.IP
	ipLen := len(currentIP)
	for leftInsertIdx <= rightInsertIdx {
		leftIP := make(net.IP, len(currentIP))
		copy(leftIP, currentIP)

		rightIP := make(net.IP, len(currentIP))
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

		if leftNet.Contains(cidr.Net.IP) {
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

	// If capacities are equal, then net IPs should be also equal
	// Otherwise networks are not the same
	if ourOnes == theirOnes {
		return n.Net.IP.Equal(cidr.Net.IP)
	}

	for i := range n.Net.IP {
		if n.Net.IP[i]&n.Net.Mask[i] != cidr.Net.IP[i]&cidr.Net.Mask[i]&n.Net.Mask[i] {
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
