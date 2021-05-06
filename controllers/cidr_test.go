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

	// TODO ipv6
}
