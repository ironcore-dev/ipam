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
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

var _ = Describe("Network webhook", func() {
	Context("When Network is not created", func() {
		It("Should check that invalid CR will be rejected", func() {
			testNamespaceName := createTestNamespace()
			crs := []Network{
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "empty-type",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						ID: NetworkIDFromInt64(1000),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-vxlan-1",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						ID:   NetworkIDFromBytes([]byte{0}),
						Type: CVXLANNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-vxlan-2",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						ID:   NetworkIDFromBytes([]byte{99}),
						Type: CVXLANNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-vxlan-3",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						ID:   NetworkIDFromBytes([]byte{0, 0, 0, 1}),
						Type: CVXLANNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-geneve-1",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						ID:   NetworkIDFromBytes([]byte{0}),
						Type: CGENEVENetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-geneve-2",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						ID:   NetworkIDFromBytes([]byte{99}),
						Type: CGENEVENetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-geneve-3",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						ID:   NetworkIDFromBytes([]byte{0, 0, 0, 1}),
						Type: CGENEVENetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-mpls-1",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						ID:   NetworkIDFromBytes([]byte{0}),
						Type: CMPLSNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-mpls-1",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						ID:   NetworkIDFromBytes([]byte{15}),
						Type: CMPLSNetworkType,
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

			crs := []Network{
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "empty-type-and-id",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "vxlan-no-id",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						Type: CVXLANNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "vxlan-border-bottom",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						Type: CVXLANNetworkType,
						ID:   NetworkIDFromBytes([]byte{100}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "vxlan-border-top",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						Type: CVXLANNetworkType,
						ID:   NetworkIDFromBytes([]byte{255, 255, 255}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "vxlan-middle-val",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						Type: CVXLANNetworkType,
						ID:   NetworkIDFromBytes([]byte{255, 16}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "geneve-no-id",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						Type: CGENEVENetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "geneve-border-bottom",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						Type: CGENEVENetworkType,
						ID:   NetworkIDFromBytes([]byte{100}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "geneve-border-top",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						Type: CGENEVENetworkType,
						ID:   NetworkIDFromBytes([]byte{255, 255, 255}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "geneve-middle-val",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						Type: CGENEVENetworkType,
						ID:   NetworkIDFromBytes([]byte{255, 16}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "mpls-no-id",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						Type: CMPLSNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "mpls-border-num",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						Type: CMPLSNetworkType,
						ID:   NetworkIDFromBytes([]byte{16}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "mpls-big-num",
						Namespace: testNamespaceName,
					},
					Spec: NetworkSpec{
						Type: CMPLSNetworkType,
						ID:   NetworkIDFromBytes([]byte{1, 11, 12, 13, 14, 15, 16}),
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

			cr := Network{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "network-with-id-and-type-failed-to-update",
					Namespace: testNamespaceName,
				},
				Spec: NetworkSpec{
					Type: CMPLSNetworkType,
					ID:   NetworkIDFromBytes([]byte{1, 11, 12, 13, 14, 15, 16}),
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
				if err != nil {
					return false
				}
				return true
			}).Should(BeTrue())

			By("Try to update network CR")
			cr.Spec.ID = NetworkIDFromInt64(10)
			Expect(k8sClient.Update(ctx, &cr)).ShouldNot(Succeed())
		})
	})

	Context("When Network is created with Type", func() {
		It("Should not allow to update CR", func() {
			testNamespaceName := createTestNamespace()

			cr := Network{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "network-with-type-failed-to-update",
					Namespace: testNamespaceName,
				},
				Spec: NetworkSpec{
					Type: CMPLSNetworkType,
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
				if err != nil {
					return false
				}
				return true
			}).Should(BeTrue())

			By("Try to update network CR")
			cr.Spec.ID = NetworkIDFromInt64(10)
			Expect(k8sClient.Update(ctx, &cr)).ShouldNot(Succeed())
		})
	})

	Context("When Network is created without ID and Type", func() {
		It("Should allow to update CR with ID and Type", func() {
			testNamespaceName := createTestNamespace()

			By("Create network CR")
			cr := Network{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "network-without-id-and-type-succeed-to-update",
					Namespace: testNamespaceName,
				},
				Spec: NetworkSpec{},
			}

			Expect(k8sClient.Create(ctx, &cr)).Should(Succeed())
			Eventually(func() bool {
				namespacedName := types.NamespacedName{
					Namespace: cr.Namespace,
					Name:      cr.Name,
				}
				err := k8sClient.Get(ctx, namespacedName, &cr)
				if err != nil {
					return false
				}
				return true
			}).Should(BeTrue())

			By("Update empty network CR ID and Type with values")
			cr.Spec.ID = NetworkIDFromBytes([]byte{1, 11, 12, 13, 14, 15, 16})
			cr.Spec.Type = CMPLSNetworkType

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
				if cr.Spec.Type != CMPLSNetworkType {
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
			cr := Network{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "blocked-network",
					Namespace: testNamespaceName,
				},
				Spec: NetworkSpec{},
			}

			namespacedName := types.NamespacedName{
				Namespace: cr.Namespace,
				Name:      cr.Name,
			}

			Expect(k8sClient.Create(ctx, &cr)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, &cr)
				if err != nil {
					return false
				}
				return true
			}).Should(BeTrue())

			By("Add reserved CIDR to status")
			cr.Status.IPv4Ranges = append(cr.Status.IPv4Ranges, *cidrMustParse("10.0.0.0/10"))
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
			cr.Status.IPv4Ranges = []CIDR{}
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
