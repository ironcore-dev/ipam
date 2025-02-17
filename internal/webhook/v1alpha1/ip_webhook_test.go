// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"fmt"

	v1alpha2 "github.com/ironcore-dev/ipam/api/ipam/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("IP webhook", func() {
	Context("When IP is not created", func() {
		It("Should check that invalid CR will be rejected", func() {
			testNamespaceName := createTestNamespace()

			crs := []v1alpha2.IP{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "without-subnet-name",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.IPSpec{},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "with-invalid-resource-ref",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.IPSpec{
						Subnet: corev1.LocalObjectReference{
							Name: "sample-subnet",
						},
						Consumer: &v1alpha2.ResourceReference{
							Kind: "",
							Name: "",
						},
					},
				},
			}

			ctx := context.Background()

			for _, cr := range crs {
				By(fmt.Sprintf("Attempting to create IP with invalid configuration %s", cr.Name))
				Expect(k8sClient.Create(ctx, &cr)).ShouldNot(Succeed())
			}
		})

		It("Should check that valid CR will be accepted", func() {
			testNamespaceName := createTestNamespace()

			crs := []v1alpha2.IP{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "with-subnet",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.IPSpec{
						Subnet: corev1.LocalObjectReference{
							Name: "sample-subnet",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "with-subnet-and-resource",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.IPSpec{
						Subnet: corev1.LocalObjectReference{
							Name: "sample-subnet",
						},
						Consumer: &v1alpha2.ResourceReference{
							Kind: "SampleKind",
							Name: "sample-name",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "with-subnet-and-ip",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.IPSpec{
						Subnet: corev1.LocalObjectReference{
							Name: "sample-subnet",
						},
						IP: v1alpha2.IPMustParse("192.168.1.1"),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "with-subnet-ip-and-resource",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.IPSpec{
						Subnet: corev1.LocalObjectReference{
							Name: "sample-subnet",
						},
						Consumer: &v1alpha2.ResourceReference{
							APIVersion: "sample.api/v1alpha1",
							Kind:       "SampleKind",
							Name:       "sample-name",
						},
						IP: v1alpha2.IPMustParse("192.168.1.1"),
					},
				},
			}

			ctx := context.Background()

			for _, cr := range crs {
				By(fmt.Sprintf("Attempting to create IP with valid configuration %s", cr.Name))
				Expect(k8sClient.Create(ctx, &cr)).Should(Succeed())
			}
		})
	})

	Context("When IP is created", func() {
		It("Should not allow to change IP or subnet", func() {
			testNamespaceName := createTestNamespace()

			crs := []v1alpha2.IP{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ip-with-subnet",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.IPSpec{
						Subnet: corev1.LocalObjectReference{
							Name: "sample-subnet",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ip-with-subnet-and-resource",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.IPSpec{
						Subnet: corev1.LocalObjectReference{
							Name: "sample-subnet",
						},
						Consumer: &v1alpha2.ResourceReference{
							Kind: "SampleKind",
							Name: "sample-name",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ip-with-subnet-and-ip",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.IPSpec{
						Subnet: corev1.LocalObjectReference{
							Name: "sample-subnet",
						},
						IP: v1alpha2.IPMustParse("192.168.1.1"),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ip-with-subnet-ip-and-resource",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha2.IPSpec{
						Subnet: corev1.LocalObjectReference{
							Name: "sample-subnet",
						},
						Consumer: &v1alpha2.ResourceReference{
							APIVersion: "sample.api/v1alpha1",
							Kind:       "SampleKind",
							Name:       "sample-name",
						},
						IP: v1alpha2.IPMustParse("192.168.1.1"),
					},
				},
			}

			ctx := context.Background()

			for _, cr := range crs {
				By(fmt.Sprintf("Сreating IP with name %s", cr.Name))
				Expect(k8sClient.Create(ctx, &cr)).Should(Succeed())

				Eventually(func() bool {
					namespacedName := types.NamespacedName{
						Namespace: cr.Namespace,
						Name:      cr.Name,
					}
					err := k8sClient.Get(ctx, namespacedName, &cr)
					return err == nil
				}, Timeout, Interval).Should(BeTrue())

				By(fmt.Sprintf("Attempting to update IP with name %s", cr.Name))
				crCopy := cr.DeepCopy()
				crCopy.Spec.IP = v1alpha2.IPMustParse("127.0.0.1")
				Expect(k8sClient.Update(ctx, crCopy)).ShouldNot(Succeed())

				crCopy = cr.DeepCopy()
				crCopy.Spec.Subnet.Name = "another-sample-subnet"
				Expect(k8sClient.Update(ctx, crCopy)).ShouldNot(Succeed())

				crCopy = cr.DeepCopy()
				crCopy.Spec.Consumer = &v1alpha2.ResourceReference{
					APIVersion: "sample.api/v1alpha1",
					Kind:       "SampleKind",
					Name:       "another-sample-name",
				}
				Expect(k8sClient.Update(ctx, crCopy)).Should(Succeed())
			}
		})
	})
})
