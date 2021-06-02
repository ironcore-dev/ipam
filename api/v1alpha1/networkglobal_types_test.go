package v1alpha1

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetworkGlobal operations", func() {
	cidrMustParse := func(s string) *CIDR {
		cidr, err := CIDRFromString(s)
		Expect(err).NotTo(HaveOccurred())
		return cidr
	}

	networkGlobalFromCidrs := func(cidrStrings ...string) *NetworkGlobal {
		cidrs := make([]CIDR, len(cidrStrings))
		for i, cidrString := range cidrStrings {
			cidrs[i] = *cidrMustParse(cidrString)
		}

		return &NetworkGlobal{
			Status: NetworkGlobalStatus{
				Ranges: cidrs,
			},
		}
	}

	Context("When Subnet is reserved on NetworkGlobal", func() {
		It("Should update list of reserved Subnets", func() {
			testCases := []struct {
				networkGlobal  *NetworkGlobal
				cidrToReserve  *CIDR
				resultingCidrs []CIDR
			}{
				{
					networkGlobal: networkGlobalFromCidrs("192.168.0.0/24", "192.168.2.0/24"),
					cidrToReserve: cidrMustParse("10.0.0.0/8"),
					resultingCidrs: []CIDR{*cidrMustParse("10.0.0.0/8"), *cidrMustParse("192.168.0.0/24"),
						*cidrMustParse("192.168.2.0/24")},
				},
				{
					networkGlobal: networkGlobalFromCidrs("192.168.0.0/24", "192.168.2.0/24"),
					cidrToReserve: cidrMustParse("200.0.0.0/8"),
					resultingCidrs: []CIDR{*cidrMustParse("192.168.0.0/24"), *cidrMustParse("192.168.2.0/24"),
						*cidrMustParse("200.0.0.0/8")},
				},
				{
					networkGlobal: networkGlobalFromCidrs("192.168.0.0/24", "192.168.2.0/24"),
					cidrToReserve: cidrMustParse("192.167.255.255/32"),
					resultingCidrs: []CIDR{*cidrMustParse("192.167.255.255/32"), *cidrMustParse("192.168.0.0/24"),
						*cidrMustParse("192.168.2.0/24")},
				},
				{
					networkGlobal: networkGlobalFromCidrs("192.168.0.0/24", "192.168.2.0/24"),
					cidrToReserve: cidrMustParse("192.168.1.0/24"),
					resultingCidrs: []CIDR{*cidrMustParse("192.168.0.0/24"), *cidrMustParse("192.168.1.0/24"),
						*cidrMustParse("192.168.2.0/24")},
				},
				{
					networkGlobal: networkGlobalFromCidrs("192.168.0.0/24", "192.168.2.0/24"),
					cidrToReserve: cidrMustParse("192.168.3.0/25"),
					resultingCidrs: []CIDR{*cidrMustParse("192.168.0.0/24"), *cidrMustParse("192.168.2.0/24"),
						*cidrMustParse("192.168.3.0/25")},
				},
				{
					networkGlobal:  networkGlobalFromCidrs(),
					cidrToReserve:  cidrMustParse("0.0.0.0/0"),
					resultingCidrs: []CIDR{*cidrMustParse("0.0.0.0/0")},
				},
			}

			for i, testCase := range testCases {
				By(fmt.Sprintf("Reserving %s in %d", testCase.cidrToReserve.String(), i))
				Expect(testCase.networkGlobal.CanReserve(testCase.cidrToReserve)).To(BeTrue())
				Expect(testCase.networkGlobal.CanRelease(testCase.cidrToReserve)).To(BeFalse())
				Expect(testCase.networkGlobal.Reserve(testCase.cidrToReserve)).To(Succeed())
				Expect(testCase.networkGlobal.Status.Ranges).To(Equal(testCase.resultingCidrs))
			}
		})
	})

	Context("When it is not possible to reserve Subnet in NetworkGlobal", func() {
		It("Should return an error", func() {
			testCases := []struct {
				networkGlobal *NetworkGlobal
				cidrToReserve *CIDR
			}{
				{
					networkGlobal: networkGlobalFromCidrs("0.0.0.0/0"),
					cidrToReserve: cidrMustParse("10.0.0.0/8"),
				},
				{
					networkGlobal: networkGlobalFromCidrs("192.168.0.0/24"),
					cidrToReserve: cidrMustParse("192.168.0.0/23"),
				},
				{
					networkGlobal: networkGlobalFromCidrs("192.168.1.0/24"),
					cidrToReserve: cidrMustParse("192.168.0.0/23"),
				},
			}

			for i, testCase := range testCases {
				By(fmt.Sprintf("Trying to reserve %s in %d", testCase.cidrToReserve.String(), i))
				Expect(testCase.networkGlobal.CanReserve(testCase.cidrToReserve)).To(BeFalse())
				networkGlobalCopy := testCase.networkGlobal.DeepCopy()
				Expect(testCase.networkGlobal.Reserve(testCase.cidrToReserve)).NotTo(Succeed())
				Expect(testCase.networkGlobal).To(Equal(networkGlobalCopy))
			}
		})
	})

	Context("When Subnet is released on NetworkGlobal", func() {
		It("Should update list of reserved subnets", func() {
			testCases := []struct {
				networkGlobal  *NetworkGlobal
				cidrToRelease  *CIDR
				resultingCidrs []CIDR
			}{
				{
					networkGlobal:  networkGlobalFromCidrs("192.168.0.0/24", "192.168.1.0/24", "192.168.2.0/24"),
					cidrToRelease:  cidrMustParse("192.168.0.0/24"),
					resultingCidrs: []CIDR{*cidrMustParse("192.168.1.0/24"), *cidrMustParse("192.168.2.0/24")},
				},
				{
					networkGlobal:  networkGlobalFromCidrs("192.168.0.0/24", "192.168.1.0/24", "192.168.2.0/24"),
					cidrToRelease:  cidrMustParse("192.168.1.0/24"),
					resultingCidrs: []CIDR{*cidrMustParse("192.168.0.0/24"), *cidrMustParse("192.168.2.0/24")},
				},
				{
					networkGlobal:  networkGlobalFromCidrs("192.168.0.0/24", "192.168.1.0/24", "192.168.2.0/24"),
					cidrToRelease:  cidrMustParse("192.168.2.0/24"),
					resultingCidrs: []CIDR{*cidrMustParse("192.168.0.0/24"), *cidrMustParse("192.168.1.0/24")},
				},
				{
					networkGlobal:  networkGlobalFromCidrs("192.168.0.0/24"),
					cidrToRelease:  cidrMustParse("192.168.0.0/24"),
					resultingCidrs: []CIDR{},
				},
			}

			for i, testCase := range testCases {
				By(fmt.Sprintf("Reserving %s in %d", testCase.cidrToRelease.String(), i))
				Expect(testCase.networkGlobal.CanRelease(testCase.cidrToRelease)).To(BeTrue())
				Expect(testCase.networkGlobal.CanReserve(testCase.cidrToRelease)).To(BeFalse())
				Expect(testCase.networkGlobal.Release(testCase.cidrToRelease)).To(Succeed())
				Expect(testCase.networkGlobal.Status.Ranges).To(Equal(testCase.resultingCidrs))
			}
		})
	})

	Context("When it is not possible to release subnet", func() {
		It("Should return an error", func() {
			testCases := []struct {
				networkGlobal *NetworkGlobal
				cidrToRelease *CIDR
			}{
				{
					networkGlobal: networkGlobalFromCidrs(),
					cidrToRelease: cidrMustParse("192.168.0.0/24"),
				},
				{
					networkGlobal: networkGlobalFromCidrs("192.168.0.0/16"),
					cidrToRelease: cidrMustParse("192.168.0.0/24"),
				},
				{
					networkGlobal: networkGlobalFromCidrs("192.168.0.0/24", "192.168.2.0/24"),
					cidrToRelease: cidrMustParse("192.168.1.0/24"),
				},
			}

			for i, testCase := range testCases {
				By(fmt.Sprintf("Trying to reserve %s in %d", testCase.cidrToRelease.String(), i))
				Expect(testCase.networkGlobal.CanRelease(testCase.cidrToRelease)).To(BeFalse())
				networkGlobalCopy := testCase.networkGlobal.DeepCopy()
				Expect(testCase.networkGlobal.Release(testCase.cidrToRelease)).NotTo(Succeed())
				Expect(testCase.networkGlobal).To(Equal(networkGlobalCopy))
			}
		})
	})
})
