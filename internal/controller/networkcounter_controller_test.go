// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ironcore-dev/ipam/api/ipam/v1alpha1"
)

var _ = Describe("NetworkCounter controller", func() {
	ns := SetupTest()

	const (
		NetworkCounterName = CVXLANCounterName
		NetworkName        = "test-network"
	)

	AfterEach(func(ctx SpecContext) {
		resources := []struct {
			res   client.Object
			list  client.ObjectList
			count func(client.ObjectList) int
		}{
			{
				res:  &v1alpha1.IP{},
				list: &v1alpha1.IPList{},
				count: func(objList client.ObjectList) int {
					list := objList.(*v1alpha1.IPList)
					return len(list.Items)
				},
			},
			{
				res:  &v1alpha1.Subnet{},
				list: &v1alpha1.SubnetList{},
				count: func(objList client.ObjectList) int {
					list := objList.(*v1alpha1.SubnetList)
					return len(list.Items)
				},
			},
			{
				res:  &v1alpha1.Network{},
				list: &v1alpha1.NetworkList{},
				count: func(objList client.ObjectList) int {
					list := objList.(*v1alpha1.NetworkList)
					return len(list.Items)
				},
			},
			{
				res:  &v1alpha1.NetworkCounter{},
				list: &v1alpha1.NetworkCounterList{},
				count: func(objList client.ObjectList) int {
					list := objList.(*v1alpha1.NetworkCounterList)
					return len(list.Items)
				},
			},
		}

		for _, r := range resources {
			Expect(k8sClient.DeleteAllOf(ctx, r.res, client.InNamespace(ns.Name))).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.List(ctx, r.list)
				if err != nil {
					return false
				}
				if r.count(r.list) > 0 {
					return false
				}
				return true
			}).Should(BeTrue())
		}
	})

	It("Should trigger update of failed networks", func(ctx SpecContext) {
		By("Counter is created")
		counterSpec := v1alpha1.NewNetworkCounterSpec(v1alpha1.VXLANNetworkType)
		counterSpec.Vacant[0].Begin = v1alpha1.NetworkIDFromInt64(101)

		networkCounter := v1alpha1.NetworkCounter{
			ObjectMeta: metav1.ObjectMeta{
				Name:      NetworkCounterName,
				Namespace: ns.Name,
			},
			Spec: *counterSpec,
		}

		Expect(k8sClient.Create(ctx, &networkCounter)).Should(Succeed())

		By("Network is created")
		network := v1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Name:      NetworkName,
				Namespace: ns.Name,
			},
			Spec: v1alpha1.NetworkSpec{
				Type: v1alpha1.VXLANNetworkType,
				ID:   v1alpha1.NetworkIDFromInt64(100),
			},
		}

		Expect(k8sClient.Create(ctx, &network)).Should(Succeed())

		By("Network has failed state")
		Eventually(func() bool {
			networkNamespacedName := types.NamespacedName{
				Namespace: ns.Name,
				Name:      NetworkName,
			}
			updatedNetwork := &v1alpha1.Network{}
			err := k8sClient.Get(ctx, networkNamespacedName, updatedNetwork)
			if err != nil {
				return false
			}
			if updatedNetwork.Status.State != v1alpha1.CFailedNetworkState {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Counter is updated")
		updatedNetworkCounter := &v1alpha1.NetworkCounter{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: NetworkCounterName}, updatedNetworkCounter)).Should(Succeed())

		updatedNetworkCounter.Spec.Vacant[0].Begin = v1alpha1.NetworkIDFromInt64(100)
		Expect(k8sClient.Update(ctx, updatedNetworkCounter)).Should(Succeed())

		By("Network has ID assigned")
		Eventually(func() bool {
			networkNamespacedName := types.NamespacedName{
				Namespace: ns.Name,
				Name:      NetworkName,
			}
			updatedNetwork := &v1alpha1.Network{}
			err := k8sClient.Get(ctx, networkNamespacedName, updatedNetwork)
			if err != nil {
				return false
			}
			if updatedNetwork.Status.State != v1alpha1.CFinishedNetworkState {
				return false
			}
			if updatedNetwork.Status.Reserved == nil {
				return false
			}
			if updatedNetwork.Status.Reserved.Int64() != 100 {
				return false
			}
			return true
		}).Should(BeTrue())
	})
})
