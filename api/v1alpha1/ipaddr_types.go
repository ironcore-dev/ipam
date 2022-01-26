package v1alpha1

import (
	"encoding/json"

	"inet.af/netaddr"
)

func IPAddrFromString(ipString string) (*IPAddr, error) {
	ip, err := netaddr.ParseIP(ipString)
	if err != nil {
		return nil, err
	}
	return &IPAddr{Net: ip}, nil
}

// +kubebuilder:validation:Type=string
type IPAddr struct {
	Net netaddr.IP `json:"-"`
}

func (in IPAddr) MarshalJSON() ([]byte, error) {
	return json.Marshal(in.String())
}

func (in *IPAddr) UnmarshalJSON(b []byte) error {
	stringVal := string(b)
	if stringVal == "null" {
		return nil
	}
	if err := json.Unmarshal(b, &stringVal); err != nil {
		return err
	}
	pIP, err := netaddr.ParseIP(stringVal)
	if err != nil {
		return err
	}
	in.Net = pIP
	return nil
}

func (in *IPAddr) String() string {
	return in.Net.String()
}

func (in *IPAddr) Equal(other *IPAddr) bool {
	return in.Net.Compare(other.Net) == 0
}

func (in *IPAddr) AsCidr() *CIDR {
	var cidrRange uint8 = 32
	if in.Net.Is6() {
		cidrRange = 128
	}

	ipNet := netaddr.IPPrefixFrom(in.Net, cidrRange)
	return CIDRFromNet(ipNet)
}

// DeepCopyInto is a deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IPAddr) DeepCopyInto(out *IPAddr) {
	if in != nil {
		out.Net = in.Net
	}
}
