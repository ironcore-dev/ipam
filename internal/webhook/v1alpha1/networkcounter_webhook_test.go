// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	v1alpha2 "github.com/ironcore-dev/ipam/api/ipam/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("NetworkCounter webhook", func() {
	Context("When NetworkCounter has sibling Networks", func() {
		It("Can't be deleted", func() {
			testNamespaceName := createTestNamespace()

			By("Create NetworkCounter with reserved ID")
			networkCounter := &v1alpha2.NetworkCounter{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-nc",
					Namespace: testNamespaceName,
				},
				Spec: *v1alpha2.NewNetworkCounterSpec(v1alpha2.GENEVENetworkType),
			}

			id, err := networkCounter.Spec.Propose()
			Expect(err).ToNot(HaveOccurred())
			Expect(networkCounter.Spec.Reserve(id)).To(Succeed())

			Expect(k8sClient.Create(ctx, networkCounter)).To(Succeed())

			namespacedName := types.NamespacedName{
				Namespace: networkCounter.Namespace,
				Name:      networkCounter.Name,
			}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, networkCounter)
				return err == nil
			}, Timeout, Interval).Should(BeTrue())

			By("Try to delete NetworkCounter, should fail")
			Expect(k8sClient.Delete(ctx, networkCounter)).To(Not(Succeed()))

			By("Release ID")
			Expect(networkCounter.Spec.Release(id)).To(Succeed())
			Expect(k8sClient.Update(ctx, networkCounter)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, networkCounter)
				if err != nil {
					return false
				}
				if !networkCounter.Spec.CanReserve(id) {
					return false
				}
				return true
			}, Timeout, Interval).Should(BeTrue())

			By("NetworkCounter should be deleted")
			Expect(k8sClient.Delete(ctx, networkCounter)).To(Succeed())

		})
	})
})
