// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0
// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/onmetal/ipam/api/ipam/v1alpha1"
	resource "k8s.io/apimachinery/pkg/api/resource"
)

// SubnetStatusApplyConfiguration represents an declarative configuration of the SubnetStatus type for use
// with apply.
type SubnetStatusApplyConfiguration struct {
	Type         *v1alpha1.SubnetAddressType  `json:"type,omitempty"`
	Locality     *v1alpha1.SubnetLocalityType `json:"locality,omitempty"`
	PrefixBits   *byte                        `json:"prefixBits,omitempty"`
	Capacity     *resource.Quantity           `json:"capacity,omitempty"`
	CapacityLeft *resource.Quantity           `json:"capacityLeft,omitempty"`
	Reserved     *v1alpha1.CIDR               `json:"reserved,omitempty"`
	Vacant       []v1alpha1.CIDR              `json:"vacant,omitempty"`
	State        *v1alpha1.SubnetState        `json:"state,omitempty"`
	Message      *string                      `json:"message,omitempty"`
}

// SubnetStatusApplyConfiguration constructs an declarative configuration of the SubnetStatus type for use with
// apply.
func SubnetStatus() *SubnetStatusApplyConfiguration {
	return &SubnetStatusApplyConfiguration{}
}

// WithType sets the Type field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Type field is set to the value of the last call.
func (b *SubnetStatusApplyConfiguration) WithType(value v1alpha1.SubnetAddressType) *SubnetStatusApplyConfiguration {
	b.Type = &value
	return b
}

// WithLocality sets the Locality field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Locality field is set to the value of the last call.
func (b *SubnetStatusApplyConfiguration) WithLocality(value v1alpha1.SubnetLocalityType) *SubnetStatusApplyConfiguration {
	b.Locality = &value
	return b
}

// WithPrefixBits sets the PrefixBits field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PrefixBits field is set to the value of the last call.
func (b *SubnetStatusApplyConfiguration) WithPrefixBits(value byte) *SubnetStatusApplyConfiguration {
	b.PrefixBits = &value
	return b
}

// WithCapacity sets the Capacity field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Capacity field is set to the value of the last call.
func (b *SubnetStatusApplyConfiguration) WithCapacity(value resource.Quantity) *SubnetStatusApplyConfiguration {
	b.Capacity = &value
	return b
}

// WithCapacityLeft sets the CapacityLeft field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the CapacityLeft field is set to the value of the last call.
func (b *SubnetStatusApplyConfiguration) WithCapacityLeft(value resource.Quantity) *SubnetStatusApplyConfiguration {
	b.CapacityLeft = &value
	return b
}

// WithReserved sets the Reserved field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Reserved field is set to the value of the last call.
func (b *SubnetStatusApplyConfiguration) WithReserved(value v1alpha1.CIDR) *SubnetStatusApplyConfiguration {
	b.Reserved = &value
	return b
}

// WithVacant adds the given value to the Vacant field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Vacant field.
func (b *SubnetStatusApplyConfiguration) WithVacant(values ...v1alpha1.CIDR) *SubnetStatusApplyConfiguration {
	for i := range values {
		b.Vacant = append(b.Vacant, values[i])
	}
	return b
}

// WithState sets the State field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the State field is set to the value of the last call.
func (b *SubnetStatusApplyConfiguration) WithState(value v1alpha1.SubnetState) *SubnetStatusApplyConfiguration {
	b.State = &value
	return b
}

// WithMessage sets the Message field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Message field is set to the value of the last call.
func (b *SubnetStatusApplyConfiguration) WithMessage(value string) *SubnetStatusApplyConfiguration {
	b.Message = &value
	return b
}
