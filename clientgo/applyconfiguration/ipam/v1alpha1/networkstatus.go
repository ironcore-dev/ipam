// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0
// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/onmetal/ipam/api/ipam/v1alpha1"
	resource "k8s.io/apimachinery/pkg/api/resource"
)

// NetworkStatusApplyConfiguration represents an declarative configuration of the NetworkStatus type for use
// with apply.
type NetworkStatusApplyConfiguration struct {
	IPv4Ranges   []v1alpha1.CIDR        `json:"ipv4Ranges,omitempty"`
	IPv6Ranges   []v1alpha1.CIDR        `json:"ipv6Ranges,omitempty"`
	Reserved     *v1alpha1.NetworkID    `json:"reserved,omitempty"`
	IPv4Capacity *resource.Quantity     `json:"ipv4Capacity,omitempty"`
	IPv6Capacity *resource.Quantity     `json:"ipv6Capacity,omitempty"`
	State        *v1alpha1.NetworkState `json:"state,omitempty"`
	Message      *string                `json:"message,omitempty"`
}

// NetworkStatusApplyConfiguration constructs an declarative configuration of the NetworkStatus type for use with
// apply.
func NetworkStatus() *NetworkStatusApplyConfiguration {
	return &NetworkStatusApplyConfiguration{}
}

// WithIPv4Ranges adds the given value to the IPv4Ranges field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the IPv4Ranges field.
func (b *NetworkStatusApplyConfiguration) WithIPv4Ranges(values ...v1alpha1.CIDR) *NetworkStatusApplyConfiguration {
	for i := range values {
		b.IPv4Ranges = append(b.IPv4Ranges, values[i])
	}
	return b
}

// WithIPv6Ranges adds the given value to the IPv6Ranges field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the IPv6Ranges field.
func (b *NetworkStatusApplyConfiguration) WithIPv6Ranges(values ...v1alpha1.CIDR) *NetworkStatusApplyConfiguration {
	for i := range values {
		b.IPv6Ranges = append(b.IPv6Ranges, values[i])
	}
	return b
}

// WithReserved sets the Reserved field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Reserved field is set to the value of the last call.
func (b *NetworkStatusApplyConfiguration) WithReserved(value v1alpha1.NetworkID) *NetworkStatusApplyConfiguration {
	b.Reserved = &value
	return b
}

// WithIPv4Capacity sets the IPv4Capacity field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the IPv4Capacity field is set to the value of the last call.
func (b *NetworkStatusApplyConfiguration) WithIPv4Capacity(value resource.Quantity) *NetworkStatusApplyConfiguration {
	b.IPv4Capacity = &value
	return b
}

// WithIPv6Capacity sets the IPv6Capacity field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the IPv6Capacity field is set to the value of the last call.
func (b *NetworkStatusApplyConfiguration) WithIPv6Capacity(value resource.Quantity) *NetworkStatusApplyConfiguration {
	b.IPv6Capacity = &value
	return b
}

// WithState sets the State field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the State field is set to the value of the last call.
func (b *NetworkStatusApplyConfiguration) WithState(value v1alpha1.NetworkState) *NetworkStatusApplyConfiguration {
	b.State = &value
	return b
}

// WithMessage sets the Message field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Message field is set to the value of the last call.
func (b *NetworkStatusApplyConfiguration) WithMessage(value string) *NetworkStatusApplyConfiguration {
	b.Message = &value
	return b
}
