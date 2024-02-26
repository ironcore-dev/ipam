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
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/ironcore-dev/ipam/api/v1alpha1"
)

var _ = Describe("Subnet client", func() {
	const (
		SubnetName         = "test-subnet"
		SubnetToDeleteName = "test-subnet-to-delete"
		DeleteLabel        = "delete-label"
		SubnetNamespace    = "default"
	)

	Context("When Subnet CR is installed", func() {
		It("Should check that Subnet CR is operational with client", func() {
			By("Creating client")
			finished := make(chan bool)

			clientset, err := NewForConfig(cfg)
			Expect(err).NotTo(HaveOccurred())

			client := clientset.Subnets(SubnetNamespace)

			prefixBits := byte(24)
			subnet := &v1alpha1.Subnet{
				ObjectMeta: v1.ObjectMeta{
					Name:      SubnetName,
					Namespace: SubnetNamespace,
				},
				Spec: v1alpha1.SubnetSpec{
					PrefixBits: &prefixBits,
					ParentSubnet: corev1.LocalObjectReference{
						Name: "test-parent-subnet",
					},
					Network: corev1.LocalObjectReference{
						Name: "test-network",
					},
					Regions: []v1alpha1.Region{
						{
							Name:              "euw",
							AvailabilityZones: []string{"a"},
						},
					},
				},
			}

			By("Creating watcher")
			watcher, err := client.Watch(ctx, v1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			events := watcher.ResultChan()

			By("Creating Subnet")
			createdSubnet := &v1alpha1.Subnet{}
			go func() {
				defer GinkgoRecover()
				createdSubnet, err = client.Create(ctx, subnet, v1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(createdSubnet.Spec).Should(Equal(subnet.Spec))
				finished <- true
			}()

			event := &watch.Event{}
			Eventually(events, CTimeout, CInterval).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Added))
			eventSubnet := event.Object.(*v1alpha1.Subnet)
			Expect(eventSubnet).NotTo(BeNil())
			Expect(eventSubnet.Spec).Should(Equal(subnet.Spec))

			<-finished

			By("Updating Subnet")
			createdSubnet.Spec.Regions = []v1alpha1.Region{
				{
					Name:              "b",
					AvailabilityZones: []string{"a"},
				},
				{
					Name:              "c",
					AvailabilityZones: []string{"a"},
				},
			}
			updatedSubnet := &v1alpha1.Subnet{}
			go func() {
				defer GinkgoRecover()
				updatedSubnet, err = client.Update(ctx, createdSubnet, v1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedSubnet.Spec).Should(Equal(createdSubnet.Spec))
				finished <- true
			}()

			Eventually(events, CTimeout, CInterval).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Modified))
			eventSubnet = event.Object.(*v1alpha1.Subnet)
			Expect(eventSubnet).NotTo(BeNil())
			Expect(eventSubnet.Spec).Should(Equal(createdSubnet.Spec))

			<-finished

			By("Updating Subnet status")
			updatedSubnet.Status.Message = "test message"
			go func() {
				defer GinkgoRecover()
				statusUpdatedSubnet, err := client.UpdateStatus(ctx, updatedSubnet, v1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(statusUpdatedSubnet.Status.Message).Should(Equal(updatedSubnet.Status.Message))
				finished <- true
			}()

			Eventually(events, CTimeout, CInterval).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Modified))
			eventSubnet = event.Object.(*v1alpha1.Subnet)
			Expect(eventSubnet).NotTo(BeNil())
			Expect(eventSubnet.Status.Message).Should(Equal(updatedSubnet.Status.Message))

			<-finished

			By("Patching Subnet")
			patch := []struct {
				Op    string `json:"op"`
				Path  string `json:"path"`
				Value string `json:"value"`
			}{{
				Op:    "replace",
				Path:  "/spec/regions/1/name",
				Value: "q",
			}}

			patchData, err := json.Marshal(patch)
			Expect(err).NotTo(HaveOccurred())

			go func() {
				defer GinkgoRecover()
				patchedSubnet, err := client.Patch(ctx, SubnetName, types.JSONPatchType, patchData, v1.PatchOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(patchedSubnet.Spec.Regions[1].Name).Should(Equal(patch[0].Value))
				finished <- true
			}()

			Eventually(events, CTimeout, CInterval).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Modified))
			eventSubnet = event.Object.(*v1alpha1.Subnet)
			Expect(eventSubnet).NotTo(BeNil())
			Expect(eventSubnet.Spec.Regions[1].Name).Should(Equal(patch[0].Value))

			<-finished

			subnetToDelete := &v1alpha1.Subnet{
				ObjectMeta: v1.ObjectMeta{
					Name:      SubnetToDeleteName,
					Namespace: SubnetNamespace,
					Labels: map[string]string{
						DeleteLabel: "",
					},
				},
				Spec: v1alpha1.SubnetSpec{
					ParentSubnet: corev1.LocalObjectReference{
						Name: "test-parent-subnet",
					},
					Network: corev1.LocalObjectReference{
						Name: "test-network",
					},
					Regions: []v1alpha1.Region{
						{
							Name:              "euw",
							AvailabilityZones: []string{"a"},
						},
					},
				},
			}

			By("Creating Subnet collection")
			_, err = client.Create(ctx, subnetToDelete, v1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(events, CTimeout, CInterval).Should(Receive())

			By("Listing Subnets")
			subnetList, err := client.List(ctx, v1.ListOptions{})
			Expect(subnetList).NotTo(BeNil())
			Expect(subnetList.Items).To(HaveLen(2))

			By("Bulk deleting Subnet")
			Expect(client.DeleteCollection(ctx, v1.DeleteOptions{}, v1.ListOptions{LabelSelector: DeleteLabel})).To(Succeed())

			By("Requesting created Subnet")
			Eventually(func() bool {
				_, err = client.Get(ctx, SubnetName, v1.GetOptions{})
				if err != nil {
					return false
				}
				return true
			}, CTimeout, CInterval).Should(BeTrue())
			Eventually(func() bool {
				_, err = client.Get(ctx, SubnetToDeleteName, v1.GetOptions{})
				if err != nil {
					return false
				}
				return true
			}, CTimeout, CInterval).Should(BeFalse())

			Eventually(events, CTimeout, CInterval).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Deleted))
			eventSubnet = event.Object.(*v1alpha1.Subnet)
			Expect(eventSubnet).NotTo(BeNil())
			Expect(eventSubnet.Name).To(Equal(SubnetToDeleteName))

			By("Deleting Subnet")
			go func() {
				defer GinkgoRecover()
				err := client.Delete(ctx, SubnetName, v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				finished <- true
			}()

			Eventually(events, CTimeout, CInterval).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Deleted))
			eventSubnet = event.Object.(*v1alpha1.Subnet)
			Expect(eventSubnet).NotTo(BeNil())
			Expect(eventSubnet.Name).To(Equal(SubnetName))

			<-finished

			watcher.Stop()
			Eventually(events, CTimeout, CInterval).Should(BeClosed())
		})
	})
})
