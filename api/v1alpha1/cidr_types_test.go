package v1alpha1

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"inet.af/netaddr"
	"k8s.io/apimachinery/pkg/util/json"
)

var _ = Describe("CIDR operations", func() {
	Context("When JSON is deserialized to CIDR", func() {
		It("Should accept CIDR string", func() {
			testCases := []struct {
				cidrJSON string
				firstIP  string
				lastIP   string
			}{
				{
					cidrJSON: `"192.168.1.1/24"`,
					firstIP:  "192.168.1.0",
					lastIP:   "192.168.1.255",
				},
				{
					cidrJSON: `"192.168.1.7/30"`,
					firstIP:  "192.168.1.4",
					lastIP:   "192.168.1.7",
				},
				{
					cidrJSON: `"8.8.8.8/32"`,
					firstIP:  "8.8.8.8",
					lastIP:   "8.8.8.8",
				},
				{
					cidrJSON: `"0.0.0.0/0"`,
					firstIP:  "0.0.0.0",
					lastIP:   "255.255.255.255",
				},
				{
					cidrJSON: `"0.0.0.0/1"`,
					firstIP:  "0.0.0.0",
					lastIP:   "127.255.255.255",
				},
				{
					cidrJSON: `"::/0"`,
					firstIP:  "::",
					lastIP:   "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff",
				},
				{
					cidrJSON: `"2001:db8:1234::/48"`,
					firstIP:  "2001:db8:1234:0000:0000:0000:0000:0000",
					lastIP:   "2001:db8:1234:ffff:ffff:ffff:ffff:ffff",
				},
				{
					cidrJSON: `"2001:db8:1234::1/128"`,
					firstIP:  "2001:db8:1234:0000:0000:0000:0000:0001",
					lastIP:   "2001:db8:1234:0000:0000:0000:0000:0001",
				},
			}

			for i, testCase := range testCases {
				By(fmt.Sprintf("Deserializing CIDR string %d", i))
				cidr := CIDR{}
				Expect(json.Unmarshal([]byte(testCase.cidrJSON), &cidr)).To(Succeed())

				By(fmt.Sprintf("Comparing to expected address range %d", i))
				first, last := cidr.ToAddressRange()

				firstIP := netaddr.MustParseIP(testCase.firstIP)
				Expect(firstIP.Compare(first)).To(Equal(0))

				lastIP := netaddr.MustParseIP(testCase.lastIP)
				Expect(lastIP.Compare(last)).To(Equal(0))

			}
		})
	})

	Context("When CIDR is serialized to JSON", func() {
		It("Should produce valid CIDR string", func() {
			testCases := []struct {
				cidr         *CIDR
				expectedJSON string
			}{
				{
					cidr:         CIDRFromNet(netaddr.IPPrefixFrom(netaddr.IPv4(192, 168, 1, 0), 24)),
					expectedJSON: `"192.168.1.0/24"`,
				},
				{
					cidr:         CIDRFromNet(netaddr.IPPrefixFrom(netaddr.IPv4(0, 0, 0, 0), 0)),
					expectedJSON: `"0.0.0.0/0"`,
				},
				{
					cidr: CIDRFromNet(netaddr.IPPrefixFrom(
						netaddr.IPFrom16([16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}), 0)),
					expectedJSON: `"::/0"`,
				},
				{
					cidr: CIDRFromNet(netaddr.IPPrefixFrom(
						netaddr.IPFrom16([16]byte{0x20, 0x1, 0xd, 0xb8, 0x12, 0x34, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}), 48)),
					expectedJSON: `"2001:db8:1234::/48"`,
				},
			}

			for i, testCase := range testCases {
				By(fmt.Sprintf("Serializing CIDR %d", i))
				data, err := json.Marshal(testCase.cidr)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Comparing to expected CIDR string %d", i))
				Expect(string(data)).To(Equal(testCase.expectedJSON))
			}
		})
	})

	Context("When one CIDR is reserved from another CIDR", func() {
		It("Should return a list of vacant subnets after reservation in a form of ", func() {
			testCases := []struct {
				cidr           *CIDR
				cidrToReserve  *CIDR
				remainingCidrs []CIDR
			}{
				{
					cidr:          cidrMustParse("192.168.1.1/24"),
					cidrToReserve: cidrMustParse("192.168.1.8/30"),
					remainingCidrs: []CIDR{*cidrMustParse("192.168.1.0/29"), *cidrMustParse("192.168.1.12/30"),
						*cidrMustParse("192.168.1.16/28"), *cidrMustParse("192.168.1.32/27"),
						*cidrMustParse("192.168.1.64/26"), *cidrMustParse("192.168.1.128/25")},
				},
				{
					cidr:          cidrMustParse("192.168.1.1/24"),
					cidrToReserve: cidrMustParse("192.168.1.0/30"),
					remainingCidrs: []CIDR{*cidrMustParse("192.168.1.4/30"), *cidrMustParse("192.168.1.8/29"),
						*cidrMustParse("192.168.1.16/28"), *cidrMustParse("192.168.1.32/27"),
						*cidrMustParse("192.168.1.64/26"), *cidrMustParse("192.168.1.128/25")},
				},
				{
					cidr:          cidrMustParse("192.168.1.1/24"),
					cidrToReserve: cidrMustParse("192.168.1.252/30"),
					remainingCidrs: []CIDR{*cidrMustParse("192.168.1.0/25"), *cidrMustParse("192.168.1.128/26"),
						*cidrMustParse("192.168.1.192/27"), *cidrMustParse("192.168.1.224/28"),
						*cidrMustParse("192.168.1.240/29"), *cidrMustParse("192.168.1.248/30")},
				},
				{
					cidr:           cidrMustParse("192.168.1.1/24"),
					cidrToReserve:  cidrMustParse("192.168.1.1/24"),
					remainingCidrs: []CIDR{},
				},
				{
					cidr:           cidrMustParse("192.168.1.0/24"),
					cidrToReserve:  cidrMustParse("192.168.1.0/25"),
					remainingCidrs: []CIDR{*cidrMustParse("192.168.1.128/25")},
				},
				{
					cidr:           cidrMustParse("192.168.1.0/24"),
					cidrToReserve:  cidrMustParse("192.168.1.128/25"),
					remainingCidrs: []CIDR{*cidrMustParse("192.168.1.0/25")},
				},
				{
					cidr:           cidrMustParse("192.168.1.0/31"),
					cidrToReserve:  cidrMustParse("192.168.1.0/32"),
					remainingCidrs: []CIDR{*cidrMustParse("192.168.1.1/32")},
				},
				{
					cidr:           cidrMustParse("192.168.1.0/31"),
					cidrToReserve:  cidrMustParse("192.168.1.1/32"),
					remainingCidrs: []CIDR{*cidrMustParse("192.168.1.0/32")},
				},
				{
					cidr:           cidrMustParse("192.168.1.0/24"),
					cidrToReserve:  cidrMustParse("10.0.0.0/16"),
					remainingCidrs: []CIDR{*cidrMustParse("192.168.1.0/24")},
				},
				{
					cidr:           cidrMustParse("0.0.0.0/0"),
					cidrToReserve:  cidrMustParse("0.0.0.0/1"),
					remainingCidrs: []CIDR{*cidrMustParse("128.0.0.0/1")},
				},
			}

			for _, testCase := range testCases {
				By(fmt.Sprintf("Reserving CIDR %s from CIDR %s", testCase.cidrToReserve.String(), testCase.cidr.String()))
				Expect(testCase.cidr.CanReserve(testCase.cidr)).To(BeTrue())
				remainingCidrs := testCase.cidr.Reserve(testCase.cidrToReserve)
				Expect(remainingCidrs).To(Equal(testCase.remainingCidrs))
			}
		})
	})

	Context("When two CIDRs are parts of bigger CIDR", func() {
		It("Should be possible to join them", func() {
			testCases := []struct {
				leftCidr      *CIDR
				rightCidr     *CIDR
				resultingCidr *CIDR
			}{
				{
					leftCidr:      cidrMustParse("192.168.0.0/32"),
					rightCidr:     cidrMustParse("192.168.0.1/32"),
					resultingCidr: cidrMustParse("192.168.0.0/31"),
				},
				{
					leftCidr:      cidrMustParse("127.255.255.255/1"),
					rightCidr:     cidrMustParse("128.0.0.0/1"),
					resultingCidr: cidrMustParse("0.0.0.0/0"),
				},
				{
					leftCidr:      cidrMustParse("192.168.0.0/24"),
					rightCidr:     cidrMustParse("192.168.1.0/24"),
					resultingCidr: cidrMustParse("192.168.0.0/23"),
				},
				{
					leftCidr:      cidrMustParse("192.168.0.0/24"),
					rightCidr:     cidrMustParse("192.168.1.0/24"),
					resultingCidr: cidrMustParse("192.168.0.0/23"),
				},
				{
					leftCidr:      cidrMustParse("2001:db8:1234::/48"),
					rightCidr:     cidrMustParse("2001:db8:1235::/48"),
					resultingCidr: cidrMustParse("2001:db8:1234::/47"),
				},
				{
					leftCidr:      cidrMustParse("::/2"),
					rightCidr:     cidrMustParse("4000::/2"),
					resultingCidr: cidrMustParse("::/1"),
				},
			}

			for _, testCase := range testCases {
				By(fmt.Sprintf("Joining %s and %s", testCase.leftCidr.String(), testCase.rightCidr.String()))
				Expect(testCase.leftCidr.CanJoin(testCase.rightCidr)).To(BeTrue())
				Expect(testCase.rightCidr.CanJoin(testCase.leftCidr)).To(BeTrue())

				leftCidrCopy := testCase.leftCidr.DeepCopy()
				leftCidrCopy.Join(testCase.rightCidr)
				Expect(leftCidrCopy).To(Equal(testCase.resultingCidr))

				rightCidrCopy := testCase.rightCidr.DeepCopy()
				rightCidrCopy.Join(testCase.leftCidr)
				Expect(rightCidrCopy).To(Equal(testCase.resultingCidr))
			}
		})
	})

	Context("When two CIDRs are not parts of bigger CIDR", func() {
		It("Should not be possible to join them", func() {
			testCases := []struct {
				aCidr *CIDR
				bCidr *CIDR
			}{
				{
					aCidr: cidrMustParse("192.168.0.0/24"),
					bCidr: cidrMustParse("192.168.0.1/32"),
				},
				{
					aCidr: cidrMustParse("192.168.0.0/24"),
					bCidr: cidrMustParse("192.168.0.0/24"),
				},
				{
					aCidr: cidrMustParse("192.168.0.0/24"),
					bCidr: cidrMustParse("192.167.255.0/24"),
				},
				{
					aCidr: cidrMustParse("192.168.0.0/29"),
					bCidr: cidrMustParse("192.168.0.8/30"),
				},
				{
					aCidr: cidrMustParse("::/1"),
					bCidr: cidrMustParse("4000::/2"),
				},
				{
					aCidr: cidrMustParse("::/0"),
					bCidr: cidrMustParse("::/0"),
				},
			}

			for _, testCase := range testCases {
				By(fmt.Sprintf("Joining unjoinable %s and %s", testCase.aCidr.String(), testCase.bCidr.String()))
				Expect(testCase.aCidr.CanJoin(testCase.bCidr)).To(BeFalse())
				Expect(testCase.bCidr.CanJoin(testCase.aCidr)).To(BeFalse())

				aCidrCopy := testCase.aCidr.DeepCopy()
				aCidrCopy.Join(testCase.bCidr)
				Expect(aCidrCopy).To(Equal(testCase.aCidr))

				bCidrCopy := testCase.bCidr.DeepCopy()
				bCidrCopy.Join(testCase.aCidr)
				Expect(bCidrCopy).To(Equal(testCase.bCidr))
			}
		})
	})

	Context("When two CIDRs are compared", func() {
		It("Should not be possible to determine if they come before or after each other", func() {
			testCases := []struct {
				ourCidr   *CIDR
				theirCidr *CIDR
				ourBefore bool
				ourAfter  bool
			}{
				{
					ourCidr:   cidrMustParse("192.168.0.0/32"),
					theirCidr: cidrMustParse("192.168.0.1/32"),
					ourBefore: true,
					ourAfter:  false,
				},
				{
					ourCidr:   cidrMustParse("192.168.0.1/32"),
					theirCidr: cidrMustParse("192.168.0.0/32"),
					ourBefore: false,
					ourAfter:  true,
				},
				{
					ourCidr:   cidrMustParse("10.0.0.0/8"),
					theirCidr: cidrMustParse("192.168.0.0/24"),
					ourBefore: true,
					ourAfter:  false,
				},
				{
					ourCidr:   cidrMustParse("::/0"),
					theirCidr: cidrMustParse("4000::/2"),
					ourBefore: false,
					ourAfter:  false,
				},
				{
					ourCidr:   cidrMustParse("4000::/2"),
					theirCidr: cidrMustParse("::/0"),
					ourBefore: false,
					ourAfter:  false,
				},
				{
					ourCidr:   cidrMustParse("4000::/128"),
					theirCidr: cidrMustParse("4000::/128"),
					ourBefore: false,
					ourAfter:  false,
				},
			}

			for _, testCase := range testCases {
				By(fmt.Sprintf("Checking CIDR order for  %s and %s", testCase.ourCidr.String(), testCase.theirCidr.String()))
				Expect(testCase.ourCidr.Before(testCase.theirCidr)).To(Equal(testCase.ourBefore))
				Expect(testCase.ourCidr.After(testCase.theirCidr)).To(Equal(testCase.ourAfter))
			}
		})
	})

	Context("When two CIDRs are compared", func() {
		It("Should not be possible to determine if they are equal or not", func() {
			Expect(cidrMustParse("4000::/128").Equal(cidrMustParse("4000::/128"))).To(BeTrue())
			Expect(cidrMustParse("::/0").Equal(cidrMustParse("4000::/128"))).To(BeFalse())
			Expect(cidrMustParse("10.0.0.0/8").Equal(cidrMustParse("10.0.0.0/8"))).To(BeTrue())
			Expect(cidrMustParse("10.0.0.0/8").Equal(cidrMustParse("192.168.0.0/24"))).To(BeFalse())
			Expect(cidrMustParse("::/0").Equal(cidrMustParse("0.0.0.0/0"))).To(BeFalse())
		})
	})
})
