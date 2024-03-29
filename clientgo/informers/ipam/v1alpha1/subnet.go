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

// SubnetInformer provides access to a shared informer and lister for
// Subnets.
type SubnetInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.SubnetLister
}

type subnetInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewSubnetInformer constructs a new informer for Subnet type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewSubnetInformer(client ipam.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredSubnetInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredSubnetInformer constructs a new informer for Subnet type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredSubnetInformer(client ipam.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.IpamV1alpha1().Subnets(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.IpamV1alpha1().Subnets(namespace).Watch(context.TODO(), options)
			},
		},
		&ipamv1alpha1.Subnet{},
		resyncPeriod,
		indexers,
	)
}

func (f *subnetInformer) defaultInformer(client ipam.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredSubnetInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *subnetInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&ipamv1alpha1.Subnet{}, f.defaultInformer)
}

func (f *subnetInformer) Lister() v1alpha1.SubnetLister {
	return v1alpha1.NewSubnetLister(f.Informer().GetIndexer())
}
