// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	json "encoding/json"
	"fmt"
	"time"

	v1alpha1 "github.com/ironcore-dev/ipam/api/ipam/v1alpha1"
	ipamv1alpha1 "github.com/ironcore-dev/ipam/clientgo/applyconfiguration/ipam/v1alpha1"
	scheme "github.com/ironcore-dev/ipam/clientgo/ipam/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// NetworkCountersGetter has a method to return a NetworkCounterInterface.
// A group's client should implement this interface.
type NetworkCountersGetter interface {
	NetworkCounters(namespace string) NetworkCounterInterface
}

// NetworkCounterInterface has methods to work with NetworkCounter resources.
type NetworkCounterInterface interface {
	Create(ctx context.Context, networkCounter *v1alpha1.NetworkCounter, opts v1.CreateOptions) (*v1alpha1.NetworkCounter, error)
	Update(ctx context.Context, networkCounter *v1alpha1.NetworkCounter, opts v1.UpdateOptions) (*v1alpha1.NetworkCounter, error)
	UpdateStatus(ctx context.Context, networkCounter *v1alpha1.NetworkCounter, opts v1.UpdateOptions) (*v1alpha1.NetworkCounter, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.NetworkCounter, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.NetworkCounterList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.NetworkCounter, err error)
	Apply(ctx context.Context, networkCounter *ipamv1alpha1.NetworkCounterApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.NetworkCounter, err error)
	ApplyStatus(ctx context.Context, networkCounter *ipamv1alpha1.NetworkCounterApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.NetworkCounter, err error)
	NetworkCounterExpansion
}

// networkCounters implements NetworkCounterInterface
type networkCounters struct {
	client rest.Interface
	ns     string
}

// newNetworkCounters returns a NetworkCounters
func newNetworkCounters(c *IpamV1alpha1Client, namespace string) *networkCounters {
	return &networkCounters{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the networkCounter, and returns the corresponding networkCounter object, and an error if there is any.
func (c *networkCounters) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.NetworkCounter, err error) {
	result = &v1alpha1.NetworkCounter{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("networkcounters").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of NetworkCounters that match those selectors.
func (c *networkCounters) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.NetworkCounterList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.NetworkCounterList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("networkcounters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested networkCounters.
func (c *networkCounters) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("networkcounters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a networkCounter and creates it.  Returns the server's representation of the networkCounter, and an error, if there is any.
func (c *networkCounters) Create(ctx context.Context, networkCounter *v1alpha1.NetworkCounter, opts v1.CreateOptions) (result *v1alpha1.NetworkCounter, err error) {
	result = &v1alpha1.NetworkCounter{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("networkcounters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(networkCounter).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a networkCounter and updates it. Returns the server's representation of the networkCounter, and an error, if there is any.
func (c *networkCounters) Update(ctx context.Context, networkCounter *v1alpha1.NetworkCounter, opts v1.UpdateOptions) (result *v1alpha1.NetworkCounter, err error) {
	result = &v1alpha1.NetworkCounter{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("networkcounters").
		Name(networkCounter.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(networkCounter).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *networkCounters) UpdateStatus(ctx context.Context, networkCounter *v1alpha1.NetworkCounter, opts v1.UpdateOptions) (result *v1alpha1.NetworkCounter, err error) {
	result = &v1alpha1.NetworkCounter{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("networkcounters").
		Name(networkCounter.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(networkCounter).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the networkCounter and deletes it. Returns an error if one occurs.
func (c *networkCounters) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("networkcounters").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *networkCounters) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("networkcounters").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched networkCounter.
func (c *networkCounters) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.NetworkCounter, err error) {
	result = &v1alpha1.NetworkCounter{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("networkcounters").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied networkCounter.
func (c *networkCounters) Apply(ctx context.Context, networkCounter *ipamv1alpha1.NetworkCounterApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.NetworkCounter, err error) {
	if networkCounter == nil {
		return nil, fmt.Errorf("networkCounter provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(networkCounter)
	if err != nil {
		return nil, err
	}
	name := networkCounter.Name
	if name == nil {
		return nil, fmt.Errorf("networkCounter.Name must be provided to Apply")
	}
	result = &v1alpha1.NetworkCounter{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("networkcounters").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *networkCounters) ApplyStatus(ctx context.Context, networkCounter *ipamv1alpha1.NetworkCounterApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.NetworkCounter, err error) {
	if networkCounter == nil {
		return nil, fmt.Errorf("networkCounter provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(networkCounter)
	if err != nil {
		return nil, err
	}

	name := networkCounter.Name
	if name == nil {
		return nil, fmt.Errorf("networkCounter.Name must be provided to Apply")
	}

	result = &v1alpha1.NetworkCounter{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("networkcounters").
		Name(*name).
		SubResource("status").
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
