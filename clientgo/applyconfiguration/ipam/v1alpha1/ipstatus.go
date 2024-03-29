// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/ironcore-dev/ipam/api/ipam/v1alpha1"
)

// IPStatusApplyConfiguration represents an declarative configuration of the IPStatus type for use
// with apply.
type IPStatusApplyConfiguration struct {
	State    *v1alpha1.IPState `json:"state,omitempty"`
	Reserved *v1alpha1.IPAddr  `json:"reserved,omitempty"`
	Message  *string           `json:"message,omitempty"`
}

// IPStatusApplyConfiguration constructs an declarative configuration of the IPStatus type for use with
// apply.
func IPStatus() *IPStatusApplyConfiguration {
	return &IPStatusApplyConfiguration{}
}

// WithState sets the State field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the State field is set to the value of the last call.
func (b *IPStatusApplyConfiguration) WithState(value v1alpha1.IPState) *IPStatusApplyConfiguration {
	b.State = &value
	return b
}

// WithReserved sets the Reserved field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Reserved field is set to the value of the last call.
func (b *IPStatusApplyConfiguration) WithReserved(value v1alpha1.IPAddr) *IPStatusApplyConfiguration {
	b.Reserved = &value
	return b
}

// WithMessage sets the Message field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Message field is set to the value of the last call.
func (b *IPStatusApplyConfiguration) WithMessage(value string) *IPStatusApplyConfiguration {
	b.Message = &value
	return b
}
