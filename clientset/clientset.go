// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package clientset

import (
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"

	"github.com/ironcore-dev/ipam/clientset/v1alpha1"
)

type Clientset interface {
	IpamV1Alpha1() v1alpha1.V1Alpha1Interface
}

type clientset struct {
	v1alpha1 v1alpha1.V1Alpha1Interface
}

func (c *clientset) IpamV1Alpha1() v1alpha1.V1Alpha1Interface {
	return c.v1alpha1
}

func NewForConfig(c *rest.Config) (Clientset, error) {
	cc := *c
	if cc.RateLimiter == nil && cc.QPS > 0 {
		if cc.Burst <= 0 {
			return nil, fmt.Errorf("burst is required to be greater than 0 when RateLimiter is not set and QPS is set to greater than 0")
		}
		cc.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(cc.QPS, cc.Burst)
	}
	var cs clientset
	var err error
	cs.v1alpha1, err = v1alpha1.NewForConfig(&cc)
	if err != nil {
		return nil, err
	}

	return &cs, nil
}
