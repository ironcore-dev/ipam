package controllers

import (
	"fmt"
	"math/big"
	"net"
)

func IsIpFree(childRanges []string, excludes []string, ipStr string) (bool, error) {
	networks, err := getNetworks(childRanges)
	if err != nil {
		return false, fmt.Errorf("unable to get child ranges: %w", err)
	}
	ip := net.ParseIP(ipStr)
	_, contain := networksContainIP(networks, ip)
	return !contain && !contains(excludes, ip.String()), nil
}

func GetFirstFreeIP(rootRange string, childRanges []string, excludes []string) (string, error) {
	_, root, err := net.ParseCIDR(rootRange)
	if err != nil {
		return "", fmt.Errorf("unable to parse root range: %w", err)
	}
	// First IP in network is subnet IP
	first := root.IP
	networks, err := getNetworks(childRanges)
	if err != nil {
		return "", fmt.Errorf("unable to get child ranges: %w", err)
	}
	// Iterate over IPs util free is found
	nextIp := first
	for i := new(big.Int).Set(big.NewInt(0)); i.Cmp(addressCount(root)) < 0; i.Add(i, big.NewInt(1)) {
		// Check if IP in child subnet
		network, contain := networksContainIP(networks, nextIp)
		if contain {
			// Skip whole subnet if IP is there
			nextIp = lastIp(network)
			i.Add(i, addressCount(network))
		} else if !contains(excludes, nextIp.String()) {
			// If IP is not excluded - return result
			return nextIp.String(), nil
		}
		nextIp = incrementIP(nextIp)
	}
	return "", fmt.Errorf("unable to find free IP")
}

func getNetworks(ranges []string) ([]*net.IPNet, error) {
	// Get networks for all provided child ranges
	networks := []*net.IPNet{}
	for _, cidr := range ranges {
		_, child, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("unable to parse child range: %w", err)
		}
		networks = append(networks, child)
	}
	return networks, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func networksContainIP(networks []*net.IPNet, ip net.IP) (*net.IPNet, bool) {
	for _, network := range networks {
		if network.Contains(ip) {
			return network, true
		}
	}
	return nil, false
}

func lastIp(n *net.IPNet) net.IP {
	last := newIP(len(n.IP))
	for i := 0; i < len(n.IP); i++ {
		last[i] = n.IP[i] | ^n.Mask[i]
	}
	return last
}

func newIP(size int) net.IP {
	if size == 4 {
		return net.ParseIP("0.0.0.0").To4()
	}
	return net.ParseIP("::")
}

// Find next IP via increment
func incrementIP(ip net.IP) (result net.IP) {
	result = make([]byte, len(ip))

	carry := true
	for i := len(ip) - 1; i >= 0; i-- {
		result[i] = ip[i]
		if carry {
			result[i]++
			if result[i] != 0 {
				carry = false
			}
		}
	}
	return
}

// Get amount of addresses in network
// since IPv6 networks can be much larger than even uint64 - big.Int is used
func addressCount(network *net.IPNet) *big.Int {
	prefixLen, bits := network.Mask.Size()
	one := &big.Int{}
	one.SetUint64(uint64(1))
	return one.Lsh(one, uint(bits)-uint(prefixLen))
}
