// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"fmt"

	v1alpha2 "github.com/ironcore-dev/ipam/api/ipam/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

var _ = Describe("Network webhook", func() {
	Context("When Network is not created", func() {
		It("Should check that invalid CR will be rejected", func() {
			testNamespaceName := createTestNamespace()
			crs := []v1alpha2.Network{
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "empty-type",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						ID: v1alpha2.NetworkIDFromInt64(1000),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-vxlan-1",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						ID:   v1alpha2.NetworkIDFromBytes([]byte{0}),
						Type: v1alpha2.VXLANNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-vxlan-2",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						ID:   v1alpha2.NetworkIDFromBytes([]byte{99}),
						Type: v1alpha2.VXLANNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-vxlan-3",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						ID:   v1alpha2.NetworkIDFromBytes([]byte{0, 0, 0, 1}),
						Type: v1alpha2.VXLANNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-geneve-1",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						ID:   v1alpha2.NetworkIDFromBytes([]byte{0}),
						Type: v1alpha2.GENEVENetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-geneve-2",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						ID:   v1alpha2.NetworkIDFromBytes([]byte{99}),
						Type: v1alpha2.GENEVENetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-geneve-3",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						ID:   v1alpha2.NetworkIDFromBytes([]byte{0, 0, 0, 1}),
						Type: v1alpha2.GENEVENetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-mpls-1",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						ID:   v1alpha2.NetworkIDFromBytes([]byte{0}),
						Type: v1alpha2.MPLSNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-mpls-1",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						ID:   v1alpha2.NetworkIDFromBytes([]byte{15}),
						Type: v1alpha2.MPLSNetworkType,
					},
				},
			}

			ctx := context.Background()

			for _, cr := range crs {
				By(fmt.Sprintf("Attempting to create Network with invalid configuration %s", cr.Name))
				Expect(k8sClient.Create(ctx, &cr)).ShouldNot(Succeed())
			}
		})
	})

	Context("When Network is not created", func() {
		It("Should check that valid CR will be accepted", func() {
			testNamespaceName := createTestNamespace()

			crs := []v1alpha2.Network{
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "empty-type-and-id",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "vxlan-no-id",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						Type: v1alpha2.VXLANNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "vxlan-border-bottom",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						Type: v1alpha2.VXLANNetworkType,
						ID:   v1alpha2.NetworkIDFromBytes([]byte{100}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "vxlan-border-top",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						Type: v1alpha2.VXLANNetworkType,
						ID:   v1alpha2.NetworkIDFromBytes([]byte{255, 255, 255}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "vxlan-middle-val",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						Type: v1alpha2.VXLANNetworkType,
						ID:   v1alpha2.NetworkIDFromBytes([]byte{255, 16}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "geneve-no-id",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						Type: v1alpha2.GENEVENetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "geneve-border-bottom",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						Type: v1alpha2.GENEVENetworkType,
						ID:   v1alpha2.NetworkIDFromBytes([]byte{100}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "geneve-border-top",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						Type: v1alpha2.GENEVENetworkType,
						ID:   v1alpha2.NetworkIDFromBytes([]byte{255, 255, 255}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "geneve-middle-val",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						Type: v1alpha2.GENEVENetworkType,
						ID:   v1alpha2.NetworkIDFromBytes([]byte{255, 16}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "mpls-no-id",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						Type: v1alpha2.MPLSNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "mpls-border-num",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						Type: v1alpha2.MPLSNetworkType,
						ID:   v1alpha2.NetworkIDFromBytes([]byte{16}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "mpls-big-num",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.NetworkSpec{
						Type: v1alpha2.MPLSNetworkType,
						ID:   v1alpha2.NetworkIDFromBytes([]byte{1, 11, 12, 13, 14, 15, 16}),
					},
				},
			}

			ctx := context.Background()

			for _, cr := range crs {
				By(fmt.Sprintf("Attempting to create Network with valid configuration %s", cr.Name))
				Expect(k8sClient.Create(ctx, &cr)).Should(Succeed())
			}
		})
	})

	Context("When Network is created with ID and Type", func() {
		It("Should not allow to update CR", func() {
			testNamespaceName := createTestNamespace()

			cr := v1alpha2.Network{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "network-with-id-and-type-failed-to-update",
					Namespace: testNamespaceName,
				},
				Spec: v1alpha2.NetworkSpec{
					Type: v1alpha2.MPLSNetworkType,
					ID:   v1alpha2.NetworkIDFromBytes([]byte{1, 11, 12, 13, 14, 15, 16}),
				},
			}

			By("Create network CR")
			Expect(k8sClient.Create(ctx, &cr)).Should(Succeed())
			Eventually(func() bool {
				namespacedName := types.NamespacedName{
					Namespace: cr.Namespace,
					Name:      cr.Name,
				}
				err := k8sClient.Get(ctx, namespacedName, &cr)
				return err == nil
			}).Should(BeTrue())

			By("Try to update network CR")
			cr.Spec.ID = v1alpha2.NetworkIDFromInt64(10)
			Expect(k8sClient.Update(ctx, &cr)).ShouldNot(Succeed())
		})
	})

	Context("When Network is created with Type", func() {
		It("Should not allow to update CR", func() {
			testNamespaceName := createTestNamespace()

			cr := v1alpha2.Network{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "network-with-type-failed-to-update",
					Namespace: testNamespaceName,
				},
				Spec: v1alpha2.NetworkSpec{
					Type: v1alpha2.MPLSNetworkType,
				},
			}

			By("Create network CR")
			Expect(k8sClient.Create(ctx, &cr)).Should(Succeed())
			Eventually(func() bool {
				namespacedName := types.NamespacedName{
					Namespace: cr.Namespace,
					Name:      cr.Name,
				}
				err := k8sClient.Get(ctx, namespacedName, &cr)
				return err == nil
			}).Should(BeTrue())

			By("Try to update network CR")
			cr.Spec.ID = v1alpha2.NetworkIDFromInt64(10)
			Expect(k8sClient.Update(ctx, &cr)).ShouldNot(Succeed())
		})
	})

	Context("When Network is created without ID and Type", func() {
		It("Should allow to update CR with ID and Type", func() {
			testNamespaceName := createTestNamespace()

			By("Create network CR")
			cr := v1alpha2.Network{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "network-without-id-and-type-succeed-to-update",
					Namespace: testNamespaceName,
				},
				Spec: v1alpha2.NetworkSpec{},
			}

			Expect(k8sClient.Create(ctx, &cr)).Should(Succeed())
			Eventually(func() bool {
				namespacedName := types.NamespacedName{
					Namespace: cr.Namespace,
					Name:      cr.Name,
				}
				err := k8sClient.Get(ctx, namespacedName, &cr)
				return err == nil
			}).Should(BeTrue())

			By("Update empty network CR ID and Type with values")
			cr.Spec.ID = v1alpha2.NetworkIDFromBytes([]byte{1, 11, 12, 13, 14, 15, 16})
			cr.Spec.Type = v1alpha2.MPLSNetworkType

			Expect(k8sClient.Update(ctx, &cr)).Should(Succeed())
			Eventually(func() bool {
				namespacedName := types.NamespacedName{
					Namespace: cr.Namespace,
					Name:      cr.Name,
				}
				err := k8sClient.Get(ctx, namespacedName, &cr)
				if err != nil {
					return false
				}
				if cr.Spec.Type != v1alpha2.MPLSNetworkType {
					return false
				}
				return true
			}).Should(BeTrue())

			By("Update network CR description")
			cr.Spec.Description = "sample description"
			Expect(k8sClient.Update(ctx, &cr)).Should(Succeed())
		})
	})

	Context("When Network has siblings", func() {
		It("Can't be deleted", func() {
			testNamespaceName := createTestNamespace()

			By("Create network CR")
			cr := v1alpha2.Network{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "blocked-network",
					Namespace: testNamespaceName,
				},
				Spec: v1alpha2.NetworkSpec{},
			}

			namespacedName := types.NamespacedName{
				Namespace: cr.Namespace,
				Name:      cr.Name,
			}

			Expect(k8sClient.Create(ctx, &cr)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, &cr)
				return err == nil
			}).Should(BeTrue())

			By("Add reserved CIDR to status")
			cr.Status.IPv4Ranges = append(cr.Status.IPv4Ranges, *v1alpha2.CidrMustParse("10.0.0.0/10"))
			Expect(k8sClient.Status().Update(ctx, &cr)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, &cr)
				if err != nil {
					return false
				}
				if len(cr.Status.IPv4Ranges) < 1 {
					return false
				}
				return true
			}).Should(BeTrue())

			By("Try to delete network, should fail")
			Expect(k8sClient.Delete(ctx, &cr)).Should(Not(Succeed()))

			By("Remove reserved CIDR from status")
			cr.Status.IPv4Ranges = []v1alpha2.CIDR{}
			Expect(k8sClient.Status().Update(ctx, &cr)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, &cr)
				if err != nil {
					return false
				}
				if len(cr.Status.IPv4Ranges) > 0 {
					return false
				}
				return true
			}).Should(BeTrue())

			By("Try to delete network, should succeed")
			Expect(k8sClient.Delete(ctx, &cr)).Should(Succeed())
		})
	})
})
