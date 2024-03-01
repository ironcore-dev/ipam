// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

// ResourceReferenceApplyConfiguration represents an declarative configuration of the ResourceReference type for use
// with apply.
type ResourceReferenceApplyConfiguration struct {
	APIVersion *string `json:"apiVersion,omitempty"`
	Kind       *string `json:"kind,omitempty"`
	Name       *string `json:"name,omitempty"`
}

// ResourceReferenceApplyConfiguration constructs an declarative configuration of the ResourceReference type for use with
// apply.
func ResourceReference() *ResourceReferenceApplyConfiguration {
	return &ResourceReferenceApplyConfiguration{}
}

// WithAPIVersion sets the APIVersion field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the APIVersion field is set to the value of the last call.
func (b *ResourceReferenceApplyConfiguration) WithAPIVersion(value string) *ResourceReferenceApplyConfiguration {
	b.APIVersion = &value
	return b
}

// WithKind sets the Kind field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Kind field is set to the value of the last call.
func (b *ResourceReferenceApplyConfiguration) WithKind(value string) *ResourceReferenceApplyConfiguration {
	b.Kind = &value
	return b
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *ResourceReferenceApplyConfiguration) WithName(value string) *ResourceReferenceApplyConfiguration {
	b.Name = &value
	return b
}
