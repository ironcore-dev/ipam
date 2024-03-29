// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/ironcore-dev/ipam/api/ipam/v1alpha1"
	v1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
)

// SubnetSpecApplyConfiguration represents an declarative configuration of the SubnetSpec type for use
// with apply.
type SubnetSpecApplyConfiguration struct {
	CIDR         *v1alpha1.CIDR                       `json:"cidr,omitempty"`
	PrefixBits   *byte                                `json:"prefixBits,omitempty"`
	Capacity     *resource.Quantity                   `json:"capacity,omitempty"`
	ParentSubnet *v1.LocalObjectReference             `json:"parentSubnet,omitempty"`
	Network      *v1.LocalObjectReference             `json:"network,omitempty"`
	Regions      []RegionApplyConfiguration           `json:"regions,omitempty"`
	Consumer     *ResourceReferenceApplyConfiguration `json:"consumer,omitempty"`
}

// SubnetSpecApplyConfiguration constructs an declarative configuration of the SubnetSpec type for use with
// apply.
func SubnetSpec() *SubnetSpecApplyConfiguration {
	return &SubnetSpecApplyConfiguration{}
}

// WithCIDR sets the CIDR field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the CIDR field is set to the value of the last call.
func (b *SubnetSpecApplyConfiguration) WithCIDR(value v1alpha1.CIDR) *SubnetSpecApplyConfiguration {
	b.CIDR = &value
	return b
}

// WithPrefixBits sets the PrefixBits field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PrefixBits field is set to the value of the last call.
func (b *SubnetSpecApplyConfiguration) WithPrefixBits(value byte) *SubnetSpecApplyConfiguration {
	b.PrefixBits = &value
	return b
}

// WithCapacity sets the Capacity field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Capacity field is set to the value of the last call.
func (b *SubnetSpecApplyConfiguration) WithCapacity(value resource.Quantity) *SubnetSpecApplyConfiguration {
	b.Capacity = &value
	return b
}

// WithParentSubnet sets the ParentSubnet field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ParentSubnet field is set to the value of the last call.
func (b *SubnetSpecApplyConfiguration) WithParentSubnet(value v1.LocalObjectReference) *SubnetSpecApplyConfiguration {
	b.ParentSubnet = &value
	return b
}

// WithNetwork sets the Network field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Network field is set to the value of the last call.
func (b *SubnetSpecApplyConfiguration) WithNetwork(value v1.LocalObjectReference) *SubnetSpecApplyConfiguration {
	b.Network = &value
	return b
}

// WithRegions adds the given value to the Regions field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Regions field.
func (b *SubnetSpecApplyConfiguration) WithRegions(values ...*RegionApplyConfiguration) *SubnetSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithRegions")
		}
		b.Regions = append(b.Regions, *values[i])
	}
	return b
}

// WithConsumer sets the Consumer field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Consumer field is set to the value of the last call.
func (b *SubnetSpecApplyConfiguration) WithConsumer(value *ResourceReferenceApplyConfiguration) *SubnetSpecApplyConfiguration {
	b.Consumer = value
	return b
}
