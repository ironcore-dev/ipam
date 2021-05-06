package controllers

import (
	. "github.com/onsi/gomega"
	"testing"
)

func TestCidr(t *testing.T) {
	RegisterTestingT(t)

	ip1, err := GetFirstFreeIP("10.12.34.0/24", []string{"10.12.34.0/26", "10.12.34.128/25"}, []string{})
	Expect(err).NotTo(HaveOccurred())
	Expect(ip1).To(Equal("10.12.34.64"))

	ip2, err := GetFirstFreeIP("10.12.34.0/24", []string{"10.12.34.0/26", "10.12.34.128/25"}, []string{"10.12.34.64", "10.12.34.65", "10.12.34.66"})
	Expect(err).NotTo(HaveOccurred())
	Expect(ip2).To(Equal("10.12.34.67"))

	// Valid
	free1, err := IsIpFree([]string{"10.12.34.0/26", "10.12.34.128/25"}, []string{}, "10.12.34.64")
	Expect(err).NotTo(HaveOccurred())
	Expect(free1).To(Equal(true))

	free2, err := IsIpFree([]string{"10.12.34.0/26", "10.12.34.128/25"}, []string{}, "10.12.34.100")
	Expect(err).NotTo(HaveOccurred())
	Expect(free2).To(Equal(true))

	// In range
	free3, err := IsIpFree([]string{"10.12.34.0/26", "10.12.34.128/25"}, []string{}, "10.12.34.127")
	Expect(err).NotTo(HaveOccurred())
	Expect(free3).To(Equal(true))

	// Out of open range
	free4, err := IsIpFree([]string{"10.12.34.0/26", "10.12.34.128/25"}, []string{}, "10.12.34.128")
	Expect(err).NotTo(HaveOccurred())
	Expect(free4).To(Equal(false))

	// Excluded
	free5, err := IsIpFree([]string{"10.12.34.0/26", "10.12.34.128/25"}, []string{"10.12.34.70"}, "10.12.34.70")
	Expect(err).NotTo(HaveOccurred())
	Expect(free5).To(Equal(false))

	// IPv6
	ipv61, err := GetFirstFreeIP("2001:db8:123:4567:89ab:cdef:1234:5678/100", []string{"2001:db8:123:4567:89ab:cdef:1234:5600/120"}, []string{})
	Expect(err).NotTo(HaveOccurred())
	Expect(ipv61).To(Equal("2001:db8:123:4567:89ab:cdef:1000:0"))

	ipv62, err := GetFirstFreeIP("2001:db8:123:4567:89ab:cdef:1234:5678/100", []string{"2001:db8:123:4567:89ab:cdef:1234:5600/120"}, []string{"2001:db8:123:4567:89ab:cdef:1000:0"})
	Expect(err).NotTo(HaveOccurred())
	Expect(ipv62).To(Equal("2001:db8:123:4567:89ab:cdef:1000:1"))

	freeipv61, err := IsIpFree([]string{"2001:db8:123:4567:89ab:cdef:1234:5600/120"}, []string{}, "2001:db8:123:4567:89ab:cdef:1000:1")
	Expect(err).NotTo(HaveOccurred())
	Expect(freeipv61).To(Equal(true))

	freeipv62, err := IsIpFree([]string{"2001:db8:123:4567:89ab:cdef:1234:5600/120"}, []string{}, "2001:db8:123:4567:89ab:cdef:1234:5601")
	Expect(err).NotTo(HaveOccurred())
	Expect(freeipv62).To(Equal(false))

	freeipv63, err := IsIpFree([]string{"2001:db8:123:4567:89ab:cdef:1234:5600/120"}, []string{"2001:db8:123:4567:89ab:cdef:1000:1"}, "2001:db8:123:4567:89ab:cdef:1000:1")
	Expect(err).NotTo(HaveOccurred())
	Expect(freeipv63).To(Equal(false))
}
