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
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/ironcore-dev/ipam/api/v1alpha1"
)

const (
	CStatusSubresource = "status"
)

type V1Alpha1Interface interface {
	IPs(namespace string) IPInterface
	Networks(namespace string) NetworkInterface
	NetworkCounters(namespace string) NetworkCounterInterface
	Subnets(namespace string) SubnetInterface
}

type v1Alpha1Client struct {
	restClient rest.Interface
}

func NewForConfig(c *rest.Config) (V1Alpha1Interface, error) {
	config := *c
	config.ContentConfig.GroupVersion = &v1alpha1.GroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &v1Alpha1Client{restClient: client}, nil
}

func (c *v1Alpha1Client) IPs(namespace string) IPInterface {
	return &ipClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}

func (c *v1Alpha1Client) Networks(namespace string) NetworkInterface {
	return &networkClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}

func (c *v1Alpha1Client) NetworkCounters(namespace string) NetworkCounterInterface {
	return &networkCounterClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}

func (c *v1Alpha1Client) Subnets(namespace string) SubnetInterface {
	return &subnetClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}
