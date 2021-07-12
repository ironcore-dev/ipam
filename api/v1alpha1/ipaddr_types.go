package v1alpha1

import (
	"net"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/json"
)

func IPAddrFromString(ipString string) (*IPAddr, error) {
	ip := net.ParseIP(ipString)

	if ip == nil {
		err := errors.New("unable to parse string to IP address")
		return nil, err
	}

	return &IPAddr{
		Net: ip,
	}, nil
}

// +kubebuilder:validation:Type=string
type IPAddr struct {
	Net net.IP `json:"-"`
}

func (in IPAddr) MarshalJSON() ([]byte, error) {
	ip := in.String()
	return json.Marshal(ip)
}

func (in *IPAddr) UnmarshalJSON(b []byte) error {
	stringVal := string(b)
	if stringVal == "null" {
		return nil
	}
	if err := json.Unmarshal(b, &stringVal); err != nil {
		return err
	}
	pIP := net.ParseIP(stringVal)

	if pIP == nil {
		err := errors.New("unable to parse string to IP address")
		return err
	}

	in.Net = pIP

	return nil
}

func (in *IPAddr) String() string {
	return in.Net.String()
}

func (in *IPAddr) Equal(other *IPAddr) bool {
	return in.Net.Equal(other.Net)
}

func (in *IPAddr) AsCidr() *CIDR {
	cidrRange := 32
	if in.Net.To4() == nil {
		cidrRange = 128
	}
	ipNet := &net.IPNet{
		IP:   in.Net,
		Mask: net.CIDRMask(cidrRange, cidrRange),
	}
	return CIDRFromNet(ipNet)
}
