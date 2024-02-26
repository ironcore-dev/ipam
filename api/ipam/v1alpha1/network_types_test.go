// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/inf.v0"
)

var _ = Describe("Network operations", func() {
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

	Context("When IPv4 and IPv6 subnets are booked in the same network", func() {
		It("Should reserve both IPv4 and IPv6 subnets", func() {
			network := networkFromCidrs()
			v4Cidr := cidrMustParse("192.168.0.0/24")
			v6Cidr := cidrMustParse("2002::1234:abcd:ffff:c0a8:101/64")

			By("Reserve CIDRs in network")
			Expect(network.CanReserve(v4Cidr)).To(BeTrue())
			Expect(network.Reserve(v4Cidr)).To(Succeed())
			Expect(network.CanReserve(v6Cidr)).To(BeTrue())
			Expect(network.Reserve(v6Cidr)).To(Succeed())

			Expect(network.Status.IPv4Capacity.AsDec().Cmp(inf.NewDecBig(v4Cidr.AddressCapacity(), 0))).To(Equal(0))
			Expect(network.Status.IPv4Ranges).To(HaveLen(1))
			Expect(network.Status.IPv4Ranges[0].Equal(v4Cidr)).To(BeTrue())
			Expect(network.Status.IPv6Capacity.AsDec().Cmp(inf.NewDecBig(v6Cidr.AddressCapacity(), 0))).To(Equal(0))
			Expect(network.Status.IPv6Ranges).To(HaveLen(1))
			Expect(network.Status.IPv6Ranges[0].Equal(v6Cidr)).To(BeTrue())

			By("Release CIDRs from network")
			Expect(network.CanRelease(v4Cidr)).To(BeTrue())
			Expect(network.Release(v4Cidr)).To(Succeed())
			Expect(network.CanRelease(v6Cidr)).To(BeTrue())
			Expect(network.Release(v6Cidr)).To(Succeed())

			Expect(network.Status.IPv4Ranges).To(HaveLen(0))
			Expect(network.Status.IPv4Capacity.IsZero()).To(BeTrue())
			Expect(network.Status.IPv6Ranges).To(HaveLen(0))
			Expect(network.Status.IPv6Capacity.IsZero()).To(BeTrue())
		})
	})
})
