// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0
// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/onmetal/ipam/api/ipam/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// NetworkLister helps list Networks.
// All objects returned here must be treated as read-only.
type NetworkLister interface {
	// List lists all Networks in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Network, err error)
	// Networks returns an object that can list and get Networks.
	Networks(namespace string) NetworkNamespaceLister
	NetworkListerExpansion
}

// networkLister implements the NetworkLister interface.
type networkLister struct {
	indexer cache.Indexer
}

// NewNetworkLister returns a new NetworkLister.
func NewNetworkLister(indexer cache.Indexer) NetworkLister {
	return &networkLister{indexer: indexer}
}

// List lists all Networks in the indexer.
func (s *networkLister) List(selector labels.Selector) (ret []*v1alpha1.Network, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Network))
	})
	return ret, err
}

// Networks returns an object that can list and get Networks.
func (s *networkLister) Networks(namespace string) NetworkNamespaceLister {
	return networkNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// NetworkNamespaceLister helps list and get Networks.
// All objects returned here must be treated as read-only.
type NetworkNamespaceLister interface {
	// List lists all Networks in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Network, err error)
	// Get retrieves the Network from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.Network, error)
	NetworkNamespaceListerExpansion
}

// networkNamespaceLister implements the NetworkNamespaceLister
// interface.
type networkNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Networks in the indexer for a given namespace.
func (s networkNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.Network, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Network))
	})
	return ret, err
}

// Get retrieves the Network from the indexer for a given namespace and name.
func (s networkNamespaceLister) Get(name string) (*v1alpha1.Network, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("network"), name)
	}
	return obj.(*v1alpha1.Network), nil
}
