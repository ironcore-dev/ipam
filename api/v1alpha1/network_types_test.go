package v1alpha1

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Network operations", func() {
	cidrMustParse := func(s string) *CIDR {
		cidr, err := CIDRFromString(s)
		Expect(err).NotTo(HaveOccurred())
		return cidr
	}

	networkFromCidrs := func(cidrStrings ...string) *Network {
		v4Cidrs := make([]CIDR, 0)
		v6Cidrs := make([]CIDR, 0)
		for _, cidrString := range cidrStrings {
			cidr := *cidrMustParse(cidrString)
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

	Context("When Subnet is reserved on Network", func() {
		It("Should update list of reserved Subnets", func() {
			testCases := []struct {
				network        *Network
				cidrToReserve  *CIDR
				resultingCidrs []CIDR
			}{
				{
					network:       networkFromCidrs("192.168.0.0/24", "192.168.2.0/24"),
					cidrToReserve: cidrMustParse("10.0.0.0/8"),
					resultingCidrs: []CIDR{*cidrMustParse("10.0.0.0/8"), *cidrMustParse("192.168.0.0/24"),
						*cidrMustParse("192.168.2.0/24")},
				},
				{
					network:       networkFromCidrs("192.168.0.0/24", "192.168.2.0/24"),
					cidrToReserve: cidrMustParse("200.0.0.0/8"),
					resultingCidrs: []CIDR{*cidrMustParse("192.168.0.0/24"), *cidrMustParse("192.168.2.0/24"),
						*cidrMustParse("200.0.0.0/8")},
				},
				{
					network:       networkFromCidrs("192.168.0.0/24", "192.168.2.0/24"),
					cidrToReserve: cidrMustParse("192.167.255.255/32"),
					resultingCidrs: []CIDR{*cidrMustParse("192.167.255.255/32"), *cidrMustParse("192.168.0.0/24"),
						*cidrMustParse("192.168.2.0/24")},
				},
				{
					network:       networkFromCidrs("192.168.0.0/24", "192.168.2.0/24"),
					cidrToReserve: cidrMustParse("192.168.1.0/24"),
					resultingCidrs: []CIDR{*cidrMustParse("192.168.0.0/24"), *cidrMustParse("192.168.1.0/24"),
						*cidrMustParse("192.168.2.0/24")},
				},
				{
					network:       networkFromCidrs("192.168.0.0/24", "192.168.2.0/24"),
					cidrToReserve: cidrMustParse("192.168.3.0/25"),
					resultingCidrs: []CIDR{*cidrMustParse("192.168.0.0/24"), *cidrMustParse("192.168.2.0/24"),
						*cidrMustParse("192.168.3.0/25")},
				},
				{
					network:        networkFromCidrs(),
					cidrToReserve:  cidrMustParse("0.0.0.0/0"),
					resultingCidrs: []CIDR{*cidrMustParse("0.0.0.0/0")},
				},
			}

			for i, testCase := range testCases {
				By(fmt.Sprintf("Reserving %s in %d", testCase.cidrToReserve.String(), i))
				Expect(testCase.network.CanReserve(testCase.cidrToReserve)).To(BeTrue())
				Expect(testCase.network.CanRelease(testCase.cidrToReserve)).To(BeFalse())
				Expect(testCase.network.Reserve(testCase.cidrToReserve)).To(Succeed())
				Expect(testCase.network.getRangesForCidr(testCase.cidrToReserve)).To(Equal(testCase.resultingCidrs))
			}
		})
	})

	Context("When it is not possible to reserve Subnet in Network", func() {
		It("Should return an error", func() {
			testCases := []struct {
				network       *Network
				cidrToReserve *CIDR
			}{
				{
					network:       networkFromCidrs("0.0.0.0/0"),
					cidrToReserve: cidrMustParse("10.0.0.0/8"),
				},
				{
					network:       networkFromCidrs("192.168.0.0/24"),
					cidrToReserve: cidrMustParse("192.168.0.0/23"),
				},
				{
					network:       networkFromCidrs("192.168.1.0/24"),
					cidrToReserve: cidrMustParse("192.168.0.0/23"),
				},
			}

			for i, testCase := range testCases {
				By(fmt.Sprintf("Trying to reserve %s in %d", testCase.cidrToReserve.String(), i))
				Expect(testCase.network.CanReserve(testCase.cidrToReserve)).To(BeFalse())
				networkCopy := testCase.network.DeepCopy()
				Expect(testCase.network.Reserve(testCase.cidrToReserve)).NotTo(Succeed())
				Expect(testCase.network).To(Equal(networkCopy))
			}
		})
	})

	Context("When Subnet is released on Network", func() {
		It("Should update list of reserved subnets", func() {
			testCases := []struct {
				network        *Network
				cidrToRelease  *CIDR
				resultingCidrs []CIDR
			}{
				{
					network:        networkFromCidrs("192.168.0.0/24", "192.168.1.0/24", "192.168.2.0/24"),
					cidrToRelease:  cidrMustParse("192.168.0.0/24"),
					resultingCidrs: []CIDR{*cidrMustParse("192.168.1.0/24"), *cidrMustParse("192.168.2.0/24")},
				},
				{
					network:        networkFromCidrs("192.168.0.0/24", "192.168.1.0/24", "192.168.2.0/24"),
					cidrToRelease:  cidrMustParse("192.168.1.0/24"),
					resultingCidrs: []CIDR{*cidrMustParse("192.168.0.0/24"), *cidrMustParse("192.168.2.0/24")},
				},
				{
					network:        networkFromCidrs("192.168.0.0/24", "192.168.1.0/24", "192.168.2.0/24"),
					cidrToRelease:  cidrMustParse("192.168.2.0/24"),
					resultingCidrs: []CIDR{*cidrMustParse("192.168.0.0/24"), *cidrMustParse("192.168.1.0/24")},
				},
				{
					network:        networkFromCidrs("192.168.0.0/24"),
					cidrToRelease:  cidrMustParse("192.168.0.0/24"),
					resultingCidrs: []CIDR{},
				},
			}

			for i, testCase := range testCases {
				By(fmt.Sprintf("Reserving %s in %d", testCase.cidrToRelease.String(), i))
				Expect(testCase.network.CanRelease(testCase.cidrToRelease)).To(BeTrue())
				Expect(testCase.network.CanReserve(testCase.cidrToRelease)).To(BeFalse())
				Expect(testCase.network.Release(testCase.cidrToRelease)).To(Succeed())
				Expect(testCase.network.getRangesForCidr(testCase.cidrToRelease)).To(Equal(testCase.resultingCidrs))
			}
		})
	})

	Context("When it is not possible to release subnet", func() {
		It("Should return an error", func() {
			testCases := []struct {
				network       *Network
				cidrToRelease *CIDR
			}{
				{
					network:       networkFromCidrs(),
					cidrToRelease: cidrMustParse("192.168.0.0/24"),
				},
				{
					network:       networkFromCidrs("192.168.0.0/16"),
					cidrToRelease: cidrMustParse("192.168.0.0/24"),
				},
				{
					network:       networkFromCidrs("192.168.0.0/24", "192.168.2.0/24"),
					cidrToRelease: cidrMustParse("192.168.1.0/24"),
				},
			}

			for i, testCase := range testCases {
				By(fmt.Sprintf("Trying to reserve %s in %d", testCase.cidrToRelease.String(), i))
				Expect(testCase.network.CanRelease(testCase.cidrToRelease)).To(BeFalse())
				networkCopy := testCase.network.DeepCopy()
				Expect(testCase.network.Release(testCase.cidrToRelease)).NotTo(Succeed())
				Expect(testCase.network).To(Equal(networkCopy))
			}
		})
	})
})
