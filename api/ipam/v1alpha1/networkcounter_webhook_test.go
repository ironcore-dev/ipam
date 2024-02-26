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
			networkCounter := &NetworkCounter{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-nc",
					Namespace: testNamespaceName,
				},
				Spec: *NewNetworkCounterSpec(CGENEVENetworkType),
			}

			id, err := networkCounter.Spec.Propose()
			Expect(err).To(BeNil())
			Expect(networkCounter.Spec.Reserve(id)).To(Succeed())

			Expect(k8sClient.Create(ctx, networkCounter)).To(Succeed())

			namespacedName := types.NamespacedName{
				Namespace: networkCounter.Namespace,
				Name:      networkCounter.Name,
			}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, networkCounter)
				return err == nil
			}, CTimeout, CInterval).Should(BeTrue())

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
			}, CTimeout, CInterval).Should(BeTrue())

			By("NetworkCounter should be deleted")
			Expect(k8sClient.Delete(ctx, networkCounter)).To(Succeed())

		})
	})
})
