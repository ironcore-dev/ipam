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

	"github.com/ironcore-dev/ipam/api/v1alpha1"
)

const (
	CIPsResourceType = "ips"
)

type IPInterface interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.IP, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.IPList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Create(ctx context.Context, ip *v1alpha1.IP, opts metav1.CreateOptions) (*v1alpha1.IP, error)
	Update(ctx context.Context, ip *v1alpha1.IP, opts metav1.UpdateOptions) (*v1alpha1.IP, error)
	UpdateStatus(ctx context.Context, ip *v1alpha1.IP, opts metav1.UpdateOptions) (*v1alpha1.IP, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1alpha1.IP, error)
}

type ipClient struct {
	restClient rest.Interface
	ns         string
}

func (c *ipClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.IP, error) {
	result := &v1alpha1.IP{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource(CIPsResourceType).
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(result)

	return result, err
}

func (c *ipClient) List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.IPList, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result := &v1alpha1.IPList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource(CIPsResourceType).
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)

	return result, err
}

func (c *ipClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	watcher, err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource(CIPsResourceType).
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)

	return watcher, err
}

func (c *ipClient) Create(ctx context.Context, ip *v1alpha1.IP, opts metav1.CreateOptions) (*v1alpha1.IP, error) {
	result := &v1alpha1.IP{}
	err := c.restClient.
		Post().
		Namespace(c.ns).
		Resource(CIPsResourceType).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(ip).
		Do(ctx).
		Into(result)

	return result, err
}

func (c *ipClient) Update(ctx context.Context, ip *v1alpha1.IP, opts metav1.UpdateOptions) (*v1alpha1.IP, error) {
	result := &v1alpha1.IP{}
	err := c.restClient.Put().
		Namespace(c.ns).
		Resource(CIPsResourceType).
		Name(ip.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(ip).
		Do(ctx).
		Into(result)

	return result, err
}

func (c *ipClient) UpdateStatus(ctx context.Context, ip *v1alpha1.IP, opts metav1.UpdateOptions) (*v1alpha1.IP, error) {
	result := &v1alpha1.IP{}
	err := c.restClient.Put().
		Namespace(c.ns).
		Resource(CIPsResourceType).
		Name(ip.Name).
		SubResource(CStatusSubresource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(ip).
		Do(ctx).
		Into(result)

	return result, err
}

func (c *ipClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.restClient.Delete().
		Namespace(c.ns).
		Resource(CIPsResourceType).
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *ipClient) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}

	return c.restClient.Delete().
		Namespace(c.ns).
		Resource(CIPsResourceType).
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *ipClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1alpha1.IP, error) {
	result := &v1alpha1.IP{}
	err := c.restClient.Patch(pt).
		Namespace(c.ns).
		Resource(CIPsResourceType).
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)

	return result, err
}
