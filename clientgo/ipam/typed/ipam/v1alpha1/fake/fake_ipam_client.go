// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/ironcore-dev/ipam/clientgo/ipam/typed/ipam/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeIpamV1alpha1 struct {
	*testing.Fake
}

func (c *FakeIpamV1alpha1) IPs(namespace string) v1alpha1.IPInterface {
	return &FakeIPs{c, namespace}
}

func (c *FakeIpamV1alpha1) Networks(namespace string) v1alpha1.NetworkInterface {
	return &FakeNetworks{c, namespace}
}

func (c *FakeIpamV1alpha1) NetworkCounters(namespace string) v1alpha1.NetworkCounterInterface {
	return &FakeNetworkCounters{c, namespace}
}

func (c *FakeIpamV1alpha1) Subnets(namespace string) v1alpha1.SubnetInterface {
	return &FakeSubnets{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeIpamV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
