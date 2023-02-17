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

package clientset

import (
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"

	"github.com/onmetal/ipam/clientset/v1alpha1"
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
