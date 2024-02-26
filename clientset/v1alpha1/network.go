// Copyright 2023 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/onmetal/ipam/api/ipam/v1alpha1"
)

const (
	CNetworksResourceType = "networks"
)

type NetworkInterface interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.Network, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.NetworkList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Create(ctx context.Context, network *v1alpha1.Network, opts metav1.CreateOptions) (*v1alpha1.Network, error)
	Update(ctx context.Context, network *v1alpha1.Network, opts metav1.UpdateOptions) (*v1alpha1.Network, error)
	UpdateStatus(ctx context.Context, network *v1alpha1.Network, opts metav1.UpdateOptions) (*v1alpha1.Network, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1alpha1.Network, error)
}

type networkClient struct {
	restClient rest.Interface
	ns         string
}

func (c *networkClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.Network, error) {
	result := &v1alpha1.Network{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource(CNetworksResourceType).
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(result)

	return result, err
}

func (c *networkClient) List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.NetworkList, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result := &v1alpha1.NetworkList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource(CNetworksResourceType).
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)

	return result, err
}

func (c *networkClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	watcher, err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource(CNetworksResourceType).
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)

	return watcher, err
}

func (c *networkClient) Create(ctx context.Context, network *v1alpha1.Network, opts metav1.CreateOptions) (*v1alpha1.Network, error) {
	result := &v1alpha1.Network{}
	err := c.restClient.
		Post().
		Namespace(c.ns).
		Resource(CNetworksResourceType).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(network).
		Do(ctx).
		Into(result)

	return result, err
}

func (c *networkClient) Update(ctx context.Context, network *v1alpha1.Network, opts metav1.UpdateOptions) (*v1alpha1.Network, error) {
	result := &v1alpha1.Network{}
	err := c.restClient.Put().
		Namespace(c.ns).
		Resource(CNetworksResourceType).
		Name(network.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(network).
		Do(ctx).
		Into(result)

	return result, err
}

func (c *networkClient) UpdateStatus(ctx context.Context, network *v1alpha1.Network, opts metav1.UpdateOptions) (*v1alpha1.Network, error) {
	result := &v1alpha1.Network{}
	err := c.restClient.Put().
		Namespace(c.ns).
		Resource(CNetworksResourceType).
		Name(network.Name).
		SubResource(CStatusSubresource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(network).
		Do(ctx).
		Into(result)

	return result, err
}

func (c *networkClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.restClient.Delete().
		Namespace(c.ns).
		Resource(CNetworksResourceType).
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *networkClient) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}

	return c.restClient.Delete().
		Namespace(c.ns).
		Resource(CNetworksResourceType).
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *networkClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1alpha1.Network, error) {
	result := &v1alpha1.Network{}
	err := c.restClient.Patch(pt).
		Namespace(c.ns).
		Resource(CNetworksResourceType).
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)

	return result, err
}
