// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	ipamv1alpha1 "github.com/ironcore-dev/ipam/api/ipam/v1alpha1"
	internalinterfaces "github.com/ironcore-dev/ipam/clientgo/informers/internalinterfaces"
	ipam "github.com/ironcore-dev/ipam/clientgo/ipam"
	v1alpha1 "github.com/ironcore-dev/ipam/clientgo/listers/ipam/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// IPInformer provides access to a shared informer and lister for
// IPs.
type IPInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.IPLister
}

type iPInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewIPInformer constructs a new informer for IP type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewIPInformer(client ipam.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredIPInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredIPInformer constructs a new informer for IP type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredIPInformer(client ipam.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.IpamV1alpha1().IPs(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.IpamV1alpha1().IPs(namespace).Watch(context.TODO(), options)
			},
		},
		&ipamv1alpha1.IP{},
		resyncPeriod,
		indexers,
	)
}

func (f *iPInformer) defaultInformer(client ipam.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredIPInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *iPInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&ipamv1alpha1.IP{}, f.defaultInformer)
}

func (f *iPInformer) Lister() v1alpha1.IPLister {
	return v1alpha1.NewIPLister(f.Informer().GetIndexer())
}
