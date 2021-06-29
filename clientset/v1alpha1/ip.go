package v1alpha1

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/onmetal/ipam/api/v1alpha1"
)

const (
	CIpsResourceType = "ips"
)

type IpInterface interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.Ip, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.IpList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Create(ctx context.Context, ip *v1alpha1.Ip, opts metav1.CreateOptions) (*v1alpha1.Ip, error)
	Update(ctx context.Context, ip *v1alpha1.Ip, opts metav1.UpdateOptions) (*v1alpha1.Ip, error)
	UpdateStatus(ctx context.Context, ip *v1alpha1.Ip, opts metav1.UpdateOptions) (*v1alpha1.Ip, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1alpha1.Ip, error)
}

type ipClient struct {
	restClient rest.Interface
	ns         string
}

func (c *ipClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.Ip, error) {
	result := &v1alpha1.Ip{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource(CIpsResourceType).
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(result)

	return result, err
}

func (c *ipClient) List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.IpList, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result := &v1alpha1.IpList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource(CIpsResourceType).
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
		Resource(CIpsResourceType).
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)

	return watcher, err
}

func (c *ipClient) Create(ctx context.Context, ip *v1alpha1.Ip, opts metav1.CreateOptions) (*v1alpha1.Ip, error) {
	result := &v1alpha1.Ip{}
	err := c.restClient.
		Post().
		Namespace(c.ns).
		Resource(CIpsResourceType).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(ip).
		Do(ctx).
		Into(result)

	return result, err
}

func (c *ipClient) Update(ctx context.Context, ip *v1alpha1.Ip, opts metav1.UpdateOptions) (*v1alpha1.Ip, error) {
	result := &v1alpha1.Ip{}
	err := c.restClient.Put().
		Namespace(c.ns).
		Resource(CIpsResourceType).
		Name(ip.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(ip).
		Do(ctx).
		Into(result)

	return result, err
}

func (c *ipClient) UpdateStatus(ctx context.Context, ip *v1alpha1.Ip, opts metav1.UpdateOptions) (*v1alpha1.Ip, error) {
	result := &v1alpha1.Ip{}
	err := c.restClient.Put().
		Namespace(c.ns).
		Resource(CIpsResourceType).
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
		Resource(CIpsResourceType).
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
		Resource(CIpsResourceType).
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *ipClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1alpha1.Ip, error) {
	result := &v1alpha1.Ip{}
	err := c.restClient.Patch(pt).
		Namespace(c.ns).
		Resource(CIpsResourceType).
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)

	return result, err
}
