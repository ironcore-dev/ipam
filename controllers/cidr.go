package controllers

import (
	"fmt"
	"net"
)

func IsIpFree(childRanges []string, excludes []string, ipStr string) (bool, error) {
	networks, err := getNetworks(childRanges)
	if err != nil {
		return false, fmt.Errorf("unable to get child ranges: %w", err)
	}
	ip := net.ParseIP(ipStr)
	return !networksContainIP(networks, ip) && !contains(excludes, ip.String()), nil
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
	for i := 0; i < addressCount(root); i++ {
		if isIpFree(networks, excludes, nextIp) {
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

func isIpFree(networks []*net.IPNet, excludes []string, ip net.IP) bool {
	return !networksContainIP(networks, ip) && !contains(excludes, ip.String())
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func networksContainIP(networks []*net.IPNet, ip net.IP) bool {
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

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

func addressCount(network *net.IPNet) int {
	prefixLen, bits := network.Mask.Size()
	return 1 << (uint64(bits) - uint64(prefixLen))
}
