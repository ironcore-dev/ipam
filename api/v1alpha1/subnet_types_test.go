package v1alpha1

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Subnet operations", func() {
	cidrMustParse := func(s string) *CIDR {
		cidr, err := CIDRFromString(s)
		Expect(err).NotTo(HaveOccurred())
		return cidr
	}

	emptySubnetFromCidr := func(mainCidr string) *Subnet {
		return &Subnet{
			Spec: SubnetSpec{
				CIDR: cidrMustParse(mainCidr),
			},
			Status: SubnetStatus{
				Vacant: []CIDR{},
			},
		}
	}

	subnetFromCidrs := func(mainCidr string, cidrStrings ...string) *Subnet {
		cidrs := make([]CIDR, len(cidrStrings))
		if len(cidrStrings) == 0 {
			cidrs = append(cidrs, *cidrMustParse(mainCidr))
		} else {
			for i, cidrString := range cidrStrings {
				cidrs[i] = *cidrMustParse(cidrString)
			}
		}

		return &Subnet{
			Spec: SubnetSpec{
				CIDR: cidrMustParse(mainCidr),
			},
			Status: SubnetStatus{
				Vacant: cidrs,
			},
		}
	}

	Context("When subnet is reserved on subnet", func() {
		It("Should update list of vacant subnets", func() {
			testCases := []struct {
				subnet          *Subnet
				cidrToReserve   *CIDR
				resultingVacant []CIDR
			}{
				{
					subnet:          subnetFromCidrs("0.0.0.0/0"),
					cidrToReserve:   cidrMustParse("0.0.0.0/1"),
					resultingVacant: []CIDR{*cidrMustParse("128.0.0.0/1")},
				},
				{
					subnet:          subnetFromCidrs("0.0.0.0/0"),
					cidrToReserve:   cidrMustParse("128.0.0.0/1"),
					resultingVacant: []CIDR{*cidrMustParse("0.0.0.0/1")},
				},
				{
					subnet:          subnetFromCidrs("0.0.0.0/0"),
					cidrToReserve:   cidrMustParse("0.0.0.0/0"),
					resultingVacant: []CIDR{},
				},
				{
					subnet:        subnetFromCidrs("192.168.0.0/18"),
					cidrToReserve: cidrMustParse("192.168.0.0/24"),
					resultingVacant: []CIDR{*cidrMustParse("192.168.1.0/24"), *cidrMustParse("192.168.2.0/23"),
						*cidrMustParse("192.168.4.0/22"), *cidrMustParse("192.168.8.0/21"),
						*cidrMustParse("192.168.16.0/20"), *cidrMustParse("192.168.32.0/19")},
				},
				{
					subnet:        subnetFromCidrs("192.168.0.0/18"),
					cidrToReserve: cidrMustParse("192.168.63.0/24"),
					resultingVacant: []CIDR{*cidrMustParse("192.168.0.0/19"), *cidrMustParse("192.168.32.0/20"),
						*cidrMustParse("192.168.48.0/21"), *cidrMustParse("192.168.56.0/22"),
						*cidrMustParse("192.168.60.0/23"), *cidrMustParse("192.168.62.0/24")},
				},
				{
					subnet:        subnetFromCidrs("192.168.0.0/18"),
					cidrToReserve: cidrMustParse("192.168.17.0/24"),
					resultingVacant: []CIDR{*cidrMustParse("192.168.0.0/20"), *cidrMustParse("192.168.16.0/24"),
						*cidrMustParse("192.168.18.0/23"), *cidrMustParse("192.168.20.0/22"),
						*cidrMustParse("192.168.24.0/21"), *cidrMustParse("192.168.32.0/19")},
				},
				{
					subnet:        subnetFromCidrs("192.168.0.0/18"),
					cidrToReserve: cidrMustParse("192.168.60.0/24"),
					resultingVacant: []CIDR{*cidrMustParse("192.168.0.0/19"), *cidrMustParse("192.168.32.0/20"),
						*cidrMustParse("192.168.48.0/21"), *cidrMustParse("192.168.56.0/22"),
						*cidrMustParse("192.168.61.0/24"), *cidrMustParse("192.168.62.0/23")},
				},
				{
					subnet:          subnetFromCidrs("0.0.0.0/0", "192.168.0.0/24", "192.168.3.0/24", "192.168.5.0/24"),
					cidrToReserve:   cidrMustParse("192.168.0.0/24"),
					resultingVacant: []CIDR{*cidrMustParse("192.168.3.0/24"), *cidrMustParse("192.168.5.0/24")},
				},
				{
					subnet:          subnetFromCidrs("0.0.0.0/0", "192.168.0.0/24", "192.168.3.0/24", "192.168.5.0/24"),
					cidrToReserve:   cidrMustParse("192.168.3.0/24"),
					resultingVacant: []CIDR{*cidrMustParse("192.168.0.0/24"), *cidrMustParse("192.168.5.0/24")},
				},
				{
					subnet:          subnetFromCidrs("0.0.0.0/0", "192.168.0.0/24", "192.168.3.0/24", "192.168.5.0/24"),
					cidrToReserve:   cidrMustParse("192.168.5.0/24"),
					resultingVacant: []CIDR{*cidrMustParse("192.168.0.0/24"), *cidrMustParse("192.168.3.0/24")},
				},
				{
					subnet:          subnetFromCidrs("0.0.0.0/0", "192.168.0.0/24", "192.168.3.0/24", "192.168.5.0/24"),
					cidrToReserve:   cidrMustParse("192.168.0.0/25"),
					resultingVacant: []CIDR{*cidrMustParse("192.168.0.128/25"), *cidrMustParse("192.168.3.0/24"), *cidrMustParse("192.168.5.0/24")},
				},
				{
					subnet:          subnetFromCidrs("0.0.0.0/0", "192.168.0.0/24", "192.168.3.0/24", "192.168.5.0/24"),
					cidrToReserve:   cidrMustParse("192.168.3.64/26"),
					resultingVacant: []CIDR{*cidrMustParse("192.168.0.0/24"), *cidrMustParse("192.168.3.0/26"), *cidrMustParse("192.168.3.128/25"), *cidrMustParse("192.168.5.0/24")},
				},
				{
					subnet:          subnetFromCidrs("0.0.0.0/0", "192.168.0.0/24", "192.168.3.0/24", "192.168.5.0/24"),
					cidrToReserve:   cidrMustParse("192.168.5.192/26"),
					resultingVacant: []CIDR{*cidrMustParse("192.168.0.0/24"), *cidrMustParse("192.168.3.0/24"), *cidrMustParse("192.168.5.0/25"), *cidrMustParse("192.168.5.128/26")},
				},
			}

			for _, testCase := range testCases {
				By(fmt.Sprintf("Reserving %s in %s", testCase.cidrToReserve.String(), testCase.subnet.Spec.CIDR.String()))
				Expect(testCase.subnet.CanReserve(testCase.cidrToReserve)).To(BeTrue())
				Expect(testCase.subnet.CanRelease(testCase.cidrToReserve)).To(BeFalse())
				Expect(testCase.subnet.Reserve(testCase.cidrToReserve)).To(Succeed())
				Expect(testCase.subnet.Status.Vacant).To(Equal(testCase.resultingVacant))
			}
		})
	})

	Context("When it is not possible to reserve subnet", func() {
		It("Should return an error", func() {
			testCases := []struct {
				subnet        *Subnet
				cidrToReserve *CIDR
			}{
				{
					subnet:        subnetFromCidrs("192.168.0.0/24"),
					cidrToReserve: cidrMustParse("192.168.1.0/24"),
				},
				{
					subnet:        subnetFromCidrs("192.168.0.0/24"),
					cidrToReserve: cidrMustParse("192.167.255.0/24"),
				},
				{
					subnet:        subnetFromCidrs("192.168.0.0/24"),
					cidrToReserve: cidrMustParse("192.167.168.0/23"),
				},
				{
					subnet:        subnetFromCidrs("192.168.0.0/24"),
					cidrToReserve: cidrMustParse("::c0a8:0/121"),
				},
				{
					subnet:        subnetFromCidrs("::/0"),
					cidrToReserve: cidrMustParse("192.168.0.0/24"),
				},
			}

			for _, testCase := range testCases {
				By(fmt.Sprintf("Reservation attempt of %s in %s", testCase.cidrToReserve.String(), testCase.subnet.Spec.CIDR.String()))
				Expect(testCase.subnet.CanReserve(testCase.cidrToReserve)).To(BeFalse())
				subnetCopy := testCase.subnet.DeepCopy()
				Expect(testCase.subnet.Reserve(testCase.cidrToReserve)).NotTo(Succeed())
				Expect(testCase.subnet).To(Equal(subnetCopy))
			}
		})
	})

	Context("When subnet is released on subnet", func() {
		It("Should update list of vacant subnets", func() {
			testCases := []struct {
				subnet          *Subnet
				cidrToRelease   *CIDR
				resultingVacant []CIDR
			}{
				{
					subnet:          subnetFromCidrs("0.0.0.0/0", "128.0.0.0/1"),
					cidrToRelease:   cidrMustParse("0.0.0.0/1"),
					resultingVacant: []CIDR{*cidrMustParse("0.0.0.0/0")},
				},
				{
					subnet:          subnetFromCidrs("0.0.0.0/0", "0.0.0.0/1"),
					cidrToRelease:   cidrMustParse("128.0.0.0/1"),
					resultingVacant: []CIDR{*cidrMustParse("0.0.0.0/0")},
				},
				{
					subnet:          emptySubnetFromCidr("0.0.0.0/0"),
					cidrToRelease:   cidrMustParse("0.0.0.0/0"),
					resultingVacant: []CIDR{*cidrMustParse("0.0.0.0/0")},
				},
				{
					subnet:          subnetFromCidrs("192.168.0.0/18", "192.168.1.0/24", "192.168.2.0/23", "192.168.4.0/22", "192.168.8.0/21", "192.168.16.0/20", "192.168.32.0/19"),
					cidrToRelease:   cidrMustParse("192.168.0.0/24"),
					resultingVacant: []CIDR{*cidrMustParse("192.168.0.0/18")},
				},
				{
					subnet:          subnetFromCidrs("192.168.0.0/18", "192.168.0.0/19", "192.168.32.0/20", "192.168.48.0/21", "192.168.56.0/22", "192.168.60.0/23", "192.168.62.0/24"),
					cidrToRelease:   cidrMustParse("192.168.63.0/24"),
					resultingVacant: []CIDR{*cidrMustParse("192.168.0.0/18")},
				},
				{
					subnet:          subnetFromCidrs("192.168.0.0/18", "192.168.0.0/20", "192.168.16.0/24", "192.168.18.0/23", "192.168.20.0/22", "192.168.24.0/21", "192.168.32.0/19"),
					cidrToRelease:   cidrMustParse("192.168.17.0/24"),
					resultingVacant: []CIDR{*cidrMustParse("192.168.0.0/18")},
				},
				{
					subnet:          subnetFromCidrs("192.168.0.0/18", "192.168.0.0/19", "192.168.32.0/20", "192.168.48.0/21", "192.168.56.0/22", "192.168.61.0/24", "192.168.62.0/23"),
					cidrToRelease:   cidrMustParse("192.168.60.0/24"),
					resultingVacant: []CIDR{*cidrMustParse("192.168.0.0/18")},
				},
			}

			for _, testCase := range testCases {
				By(fmt.Sprintf("Releasing %s to %s", testCase.cidrToRelease.String(), testCase.subnet.Spec.CIDR.String()))
				Expect(testCase.subnet.CanReserve(testCase.cidrToRelease)).To(BeFalse())
				Expect(testCase.subnet.CanRelease(testCase.cidrToRelease)).To(BeTrue())
				Expect(testCase.subnet.Release(testCase.cidrToRelease)).To(Succeed())
				Expect(testCase.subnet.Status.Vacant).To(Equal(testCase.resultingVacant))
			}
		})
	})

	Context("When it is not possible to release subnet", func() {
		It("Should return an error", func() {
			testCases := []struct {
				subnet        *Subnet
				cidrToRelease *CIDR
			}{
				{
					subnet:        emptySubnetFromCidr("0.0.0.0/1"),
					cidrToRelease: cidrMustParse("192.168.63.0/24"),
				},
				{
					subnet:        emptySubnetFromCidr("0.0.0.0/1"),
					cidrToRelease: cidrMustParse("128.0.0.0/1"),
				},
				{
					subnet:        subnetFromCidrs("0.0.0.0/1"),
					cidrToRelease: cidrMustParse("0.0.0.0/1"),
				},
				{
					subnet:        subnetFromCidrs("0.0.0.0/1"),
					cidrToRelease: cidrMustParse("10.0.0.0/8"),
				},
				{
					subnet:        subnetFromCidrs("0.0.0.0/1", "10.0.0.0/8"),
					cidrToRelease: cidrMustParse("10.0.0.1/24"),
				},
			}

			for _, testCase := range testCases {
				By(fmt.Sprintf("Release attempt of %s to %s", testCase.cidrToRelease.String(), testCase.subnet.Spec.CIDR.String()))
				Expect(testCase.subnet.CanRelease(testCase.cidrToRelease)).To(BeFalse())
				subnetCopy := testCase.subnet.DeepCopy()
				Expect(testCase.subnet.Release(testCase.cidrToRelease)).NotTo(Succeed())
				Expect(testCase.subnet).To(Equal(subnetCopy))
			}
		})
	})

	Context("When Subnet spec is filled", func() {
		It("Should correctly populate state", func() {
			localCidr := cidrMustParse("0.0.0.0/1")
			localSubnet := Subnet{
				Spec: SubnetSpec{
					CIDR:              localCidr,
					Regions:           []string{"euw"},
					AvailabilityZones: []string{"a"},
				},
			}

			localSubnet.PopulateStatus()

			Expect(localSubnet.Status.Capacity.Value()).To(Equal(localCidr.AddressCapacity().Int64()))
			Expect(localSubnet.Status.CapacityLeft.Value()).To(Equal(localCidr.AddressCapacity().Int64()))
			Expect(localSubnet.Status.Locality).To(Equal(CLocalSubnetLocalityType))
			Expect(localSubnet.Status.Vacant).To(HaveLen(1))
			Expect(localSubnet.Status.Vacant[0].Equal(localCidr)).To(BeTrue())
			Expect(localSubnet.Status.Type).To(Equal(CIPv4SubnetType))
			Expect(localSubnet.Status.Message).To(BeZero())

			regionalCidr := cidrMustParse("::/1")
			regionalSubnet := Subnet{
				Spec: SubnetSpec{
					CIDR:              regionalCidr,
					Regions:           []string{"euw"},
					AvailabilityZones: []string{"a", "b"},
				},
			}

			regionalSubnet.PopulateStatus()

			Expect(regionalSubnet.Status.Capacity.Value()).To(Equal(regionalCidr.AddressCapacity().Int64()))
			Expect(regionalSubnet.Status.CapacityLeft.Value()).To(Equal(regionalCidr.AddressCapacity().Int64()))
			Expect(regionalSubnet.Status.Locality).To(Equal(CRegionalSubnetLocalityType))
			Expect(regionalSubnet.Status.Vacant).To(HaveLen(1))
			Expect(regionalSubnet.Status.Vacant[0].Equal(regionalCidr)).To(BeTrue())
			Expect(regionalSubnet.Status.Type).To(Equal(CIPv6SubnetType))
			Expect(regionalSubnet.Status.Message).To(BeZero())

			multiregionalCidr := cidrMustParse("::/1")
			multiregionalSubnet := Subnet{
				Spec: SubnetSpec{
					CIDR:              multiregionalCidr,
					Regions:           []string{"euw", "eun"},
					AvailabilityZones: []string{"a", "b"},
				},
			}

			multiregionalSubnet.PopulateStatus()

			Expect(multiregionalSubnet.Status.Capacity.Value()).To(Equal(multiregionalCidr.AddressCapacity().Int64()))
			Expect(multiregionalSubnet.Status.CapacityLeft.Value()).To(Equal(multiregionalCidr.AddressCapacity().Int64()))
			Expect(multiregionalSubnet.Status.Locality).To(Equal(CMultiregionalSubnetLocalityType))
			Expect(multiregionalSubnet.Status.Vacant).To(HaveLen(1))
			Expect(multiregionalSubnet.Status.Vacant[0].Equal(multiregionalCidr)).To(BeTrue())
			Expect(multiregionalSubnet.Status.Type).To(Equal(CIPv6SubnetType))
			Expect(multiregionalSubnet.Status.Message).To(BeZero())
		})
	})
})
