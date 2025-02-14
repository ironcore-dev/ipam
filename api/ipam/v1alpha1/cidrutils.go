// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"net/netip"

	. "github.com/onsi/gomega"
)

func IPMustParse(ipString string) *IPAddr {
	ip, err := IPAddrFromString(ipString)
	if err != nil {
		panic(err)
	}
	return ip
}

func CidrMustParse(s string) *CIDR {
	cidr, err := CIDRFromString(s)
	Expect(err).NotTo(HaveOccurred())
	return cidr
}
func SubnetFromCidrs(mainCidr string, cidrStrings ...string) *Subnet {
	cidrs := make([]CIDR, len(cidrStrings))
	if len(cidrStrings) == 0 {
		cidrs = append(cidrs, *CidrMustParse(mainCidr))
	} else {
		for i, cidrString := range cidrStrings {
			cidrs[i] = *CidrMustParse(cidrString)
		}
	}

	cidr := CidrMustParse(mainCidr)
	return &Subnet{
		Spec: SubnetSpec{
			CIDR: cidr,
		},
		Status: SubnetStatus{
			Vacant:   cidrs,
			Reserved: cidr,
		},
	}
}

func NetworkFromCidrs(cidrStrings ...string) *Network {
	v4Cidrs := make([]CIDR, 0)
	v6Cidrs := make([]CIDR, 0)
	for _, cidrString := range cidrStrings {
		cidr := *CidrMustParse(cidrString)
		if cidr.IsIPv4() {
			v4Cidrs = append(v4Cidrs, cidr)
		} else {
			v6Cidrs = append(v6Cidrs, cidr)
		}
	}

	nw := &Network{
		Status: NetworkStatus{
			IPv4Ranges: v4Cidrs,
			IPv6Ranges: v6Cidrs,
		},
	}

	return nw
}
func EmptySubnetFromCidr(mainCidr string) *Subnet {
	cidr := CidrMustParse(mainCidr)
	return &Subnet{
		Spec: SubnetSpec{
			CIDR: cidr,
		},
		Status: SubnetStatus{
			Vacant:   []CIDR{},
			Reserved: cidr,
		},
	}
}
func CIDRFromString(cidrString string) (*CIDR, error) {
	cidr, err := netip.ParsePrefix(cidrString)
	if err != nil {
		return nil, err
	}
	return &CIDR{
		Net: cidr,
	}, nil
}
