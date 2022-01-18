package v1alpha1

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"inet.af/netaddr"
	"k8s.io/apimachinery/pkg/util/json"
)

var _ = Describe("IP operations", func() {
	Context("When JSON is deserialized to IP", func() {
		It("Should accept IP string", func() {
			testCases := []string{
				`"192.168.1.1"`,
				`"8.8.8.8"`,
				`"0.0.0.0"`,
				`"::"`,
				`"2001:db8:1234::"`,
			}

			for i, testCase := range testCases {
				By(fmt.Sprintf("Deserializing IP string %d", i))
				ip := IPAddr{}
				Expect(json.Unmarshal([]byte(testCase), &ip)).To(Succeed())
			}
		})
	})

	Context("When IP is serialized to JSON", func() {
		It("Should produce valid IP string", func() {
			testCases := []struct {
				ip           *IPAddr
				expectedJSON string
			}{
				{
					ip: &IPAddr{
						Net: netaddr.IPv4(192, 168, 1, 0),
					},
					expectedJSON: `"192.168.1.0"`,
				},
				{
					ip: &IPAddr{
						Net: netaddr.IPv4(0, 0, 0, 0),
					},
					expectedJSON: `"0.0.0.0"`,
				},
				{
					ip: &IPAddr{
						Net: netaddr.IPv6Raw([16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}),
					},
					expectedJSON: `"::"`,
				},
				{
					ip: &IPAddr{
						Net: netaddr.IPv6Raw([16]byte{0x20, 0x1, 0xd, 0xb8, 0x12, 0x34, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}),
					},
					expectedJSON: `"2001:db8:1234::"`,
				},
			}

			for i, testCase := range testCases {
				By(fmt.Sprintf("Serializing IP %d", i))
				data, err := json.Marshal(testCase.ip)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Comparing to expected IP string %d", i))
				Expect(string(data)).To(Equal(testCase.expectedJSON))
			}
		})
	})

})
