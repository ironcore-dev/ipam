// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/ironcore-dev/ipam/api/ipam/v1alpha1"
)

var _ = Describe("Network controller", func() {
	ns := SetupTest()

	const (
		VXLANNetworkName  = "test-vxlan-network"
		GENEVENetworkName = "test-geneve-network"
		MPLSNetworkName   = "test-mpls-network"
		CopyPostfix       = "-copy"
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

	It("Should get ID assigned if ID is vacant", func(ctx SpecContext) {
		testNetworkCases := []struct {
			counterName string
			firstId     *v1alpha1.NetworkID
			network     *v1alpha1.Network
		}{
			{
				counterName: CVXLANCounterName,
				firstId:     v1alpha1.VXLANFirstAvaliableID,
				network: &v1alpha1.Network{
					ObjectMeta: metav1.ObjectMeta{
						Name:      VXLANNetworkName,
						Namespace: ns.Name,
					},
					Spec: v1alpha1.NetworkSpec{
						Type: v1alpha1.VXLANNetworkType,
					},
				},
			},
			{
				counterName: CGENEVECounterName,
				firstId:     v1alpha1.GENEVEFirstAvaliableID,
				network: &v1alpha1.Network{
					ObjectMeta: metav1.ObjectMeta{
						Name:      GENEVENetworkName,
						Namespace: ns.Name,
					},
					Spec: v1alpha1.NetworkSpec{
						Type: v1alpha1.GENEVENetworkType,
					},
				},
			},
			{
				counterName: CMPLSCounterName,
				firstId:     v1alpha1.MPLSFirstAvailableID,
				network: &v1alpha1.Network{
					ObjectMeta: metav1.ObjectMeta{
						Name:      MPLSNetworkName,
						Namespace: ns.Name,
					},
					Spec: v1alpha1.NetworkSpec{
						Type: v1alpha1.MPLSNetworkType,
					},
				},
			},
		}

		for _, testNetworkCase := range testNetworkCases {
			testNetwork := testNetworkCase.network

			Expect(k8sClient.Create(ctx, testNetwork)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: testNetwork.Name}, testNetwork)
				if err != nil {
					return false
				}
				if !controllerutil.ContainsFinalizer(testNetwork, CNetworkFinalizer) {
					return false
				}
				if testNetwork.Status.State != v1alpha1.CFinishedNetworkState {
					return false
				}
				if testNetwork.Status.Reserved == nil {
					return false
				}
				return true
			}).Should(BeTrue())

			By(fmt.Sprintf("%s network counter is created", testNetworkCase.network.Spec.Type))
			counter := v1alpha1.NetworkCounter{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKey{Namespace: testNetwork.Namespace, Name: testNetworkCase.counterName}, &counter)
				return err == nil
			}).Should(BeTrue())

			By(fmt.Sprintf("%s network ID reserved in counter", testNetworkCase.network.Spec.Type))
			Expect(v1alpha1.NewNetworkCounterSpec(testNetwork.Spec.Type).CanReserve(testNetwork.Status.Reserved)).Should(BeTrue())
			Expect(counter.Spec.CanReserve(testNetwork.Status.Reserved)).Should(BeFalse())

			By(fmt.Sprintf("%s network ID with the same ID is created", testNetworkCase.network.Spec.Type))
			testNetworkCopy := v1alpha1.Network{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testNetwork.Name + CopyPostfix,
					Namespace: testNetwork.Namespace,
				},
				Spec: *testNetwork.Spec.DeepCopy(),
			}
			testNetworkCopy.Spec.ID = testNetwork.Status.Reserved
			Expect(k8sClient.Create(ctx, &testNetworkCopy)).Should(Succeed())

			By(fmt.Sprintf("%s network ID with the same ID fails on ID reservation", testNetworkCase.network.Spec.Type))
			Eventually(func() bool {
				networkCopyNamespacedName := types.NamespacedName{
					Namespace: testNetworkCopy.Namespace,
					Name:      testNetworkCopy.Name,
				}
				err := k8sClient.Get(ctx, networkCopyNamespacedName, &testNetworkCopy)
				if err != nil {
					return false
				}
				if testNetworkCopy.Status.State != v1alpha1.CFailedNetworkState {
					return false
				}
				return true
			}).Should(BeTrue())

			By(fmt.Sprintf("%s network ID with the same ID deleted", testNetworkCase.network.Spec.Type))
			Expect(k8sClient.Delete(ctx, &testNetworkCopy)).Should(Succeed())

			By(fmt.Sprintf("%s network ID CR deleted", testNetworkCase.network.Spec.Type))
			oldNetworkID := testNetwork.Status.Reserved.DeepCopy()
			Expect(k8sClient.Delete(ctx, testNetwork)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: testNetwork.Name}, testNetwork)
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())

			By(fmt.Sprintf("%s network ID released", testNetworkCase.network.Spec.Type))
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKey{Namespace: testNetwork.Namespace, Name: testNetworkCase.counterName}, &counter)
				return err == nil
			}).Should(BeTrue())

			Expect(counter.Spec.CanReserve(oldNetworkID)).Should(BeTrue())
		}
	})
})
