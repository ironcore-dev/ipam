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
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/onmetal/ipam/api/ipam/v1alpha1"
)

var _ = Describe("Network client", func() {
	const (
		NetworkName         = "test-network"
		NetworkToDeleteName = "test-network-to-delete"
		DeleteLabel         = "delete-label"
		NetworkNamespace    = "default"
	)

	Context("When Network CR is installed", func() {
		It("Should check that Network CR is operational with client", func() {
			By("Creating client")
			finished := make(chan bool)

			clientset, err := NewForConfig(cfg)
			Expect(err).NotTo(HaveOccurred())

			client := clientset.Networks(NetworkNamespace)

			network := &v1alpha1.Network{
				ObjectMeta: v1.ObjectMeta{
					Name:      NetworkName,
					Namespace: NetworkNamespace,
				},
				Spec: v1alpha1.NetworkSpec{
					Description: "empty network",
				},
			}

			By("Creating watcher")
			watcher, err := client.Watch(ctx, v1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			events := watcher.ResultChan()

			By("Creating Network")
			createdNetwork := &v1alpha1.Network{}
			go func() {
				defer GinkgoRecover()
				createdNetwork, err = client.Create(ctx, network, v1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(createdNetwork.Spec).Should(Equal(network.Spec))
				finished <- true
			}()

			event := &watch.Event{}
			Eventually(events, CTimeout, CInterval).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Added))
			eventNetwork := event.Object.(*v1alpha1.Network)
			Expect(eventNetwork).NotTo(BeNil())
			Expect(eventNetwork.Spec).Should(Equal(network.Spec))

			<-finished

			By("Updating Network")
			createdNetwork.Spec.Type = v1alpha1.CGENEVENetworkType
			updatedNetwork := &v1alpha1.Network{}
			go func() {
				defer GinkgoRecover()
				updatedNetwork, err = client.Update(ctx, createdNetwork, v1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedNetwork.Spec).Should(Equal(createdNetwork.Spec))
				finished <- true
			}()

			Eventually(events, CTimeout, CInterval).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Modified))
			eventNetwork = event.Object.(*v1alpha1.Network)
			Expect(eventNetwork).NotTo(BeNil())
			Expect(eventNetwork.Spec).Should(Equal(createdNetwork.Spec))

			<-finished

			By("Updating Network status")
			updatedNetwork.Status.Message = "test message"
			go func() {
				defer GinkgoRecover()
				statusUpdatedNetwork, err := client.UpdateStatus(ctx, updatedNetwork, v1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(statusUpdatedNetwork.Status.Message).Should(Equal(updatedNetwork.Status.Message))
				finished <- true
			}()

			Eventually(events, CTimeout, CInterval).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Modified))
			eventNetwork = event.Object.(*v1alpha1.Network)
			Expect(eventNetwork).NotTo(BeNil())
			Expect(eventNetwork.Status.Message).Should(Equal(updatedNetwork.Status.Message))

			<-finished

			By("Patching Network")
			patch := []struct {
				Op    string `json:"op"`
				Path  string `json:"path"`
				Value string `json:"value"`
			}{{
				Op:    "replace",
				Path:  "/spec/type",
				Value: string(v1alpha1.CVXLANNetworkType),
			}}

			patchData, err := json.Marshal(patch)
			Expect(err).NotTo(HaveOccurred())

			go func() {
				defer GinkgoRecover()
				patchedNetwork, err := client.Patch(ctx, NetworkName, types.JSONPatchType, patchData, v1.PatchOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(patchedNetwork.Spec.Type).Should(BeEquivalentTo(patch[0].Value))
				finished <- true
			}()

			Eventually(events, CTimeout, CInterval).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Modified))
			eventNetwork = event.Object.(*v1alpha1.Network)
			Expect(eventNetwork).NotTo(BeNil())
			Expect(eventNetwork.Spec.Type).Should(BeEquivalentTo(patch[0].Value))

			<-finished

			networkToDelete := &v1alpha1.Network{
				ObjectMeta: v1.ObjectMeta{
					Name:      NetworkToDeleteName,
					Namespace: NetworkNamespace,
					Labels: map[string]string{
						DeleteLabel: "",
					},
				},
				Spec: v1alpha1.NetworkSpec{
					Description: "to delete",
				},
			}

			By("Creating Network collection")
			_, err = client.Create(ctx, networkToDelete, v1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(events, CTimeout, CInterval).Should(Receive())

			By("Listing Networks")
			networkList, err := client.List(ctx, v1.ListOptions{})
			Expect(networkList).NotTo(BeNil())
			Expect(networkList.Items).To(HaveLen(2))

			By("Bulk deleting Network")
			Expect(client.DeleteCollection(ctx, v1.DeleteOptions{}, v1.ListOptions{LabelSelector: DeleteLabel})).To(Succeed())

			By("Requesting created Network")
			Eventually(func() bool {
				_, err = client.Get(ctx, NetworkName, v1.GetOptions{})
				return err == nil
			}, CTimeout, CInterval).Should(BeTrue())
			Eventually(func() bool {
				_, err = client.Get(ctx, NetworkToDeleteName, v1.GetOptions{})
				return err == nil
			}, CTimeout, CInterval).Should(BeFalse())

			Eventually(events, CTimeout, CInterval).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Deleted))
			eventNetwork = event.Object.(*v1alpha1.Network)
			Expect(eventNetwork).NotTo(BeNil())
			Expect(eventNetwork.Name).To(Equal(NetworkToDeleteName))

			By("Deleting Network")
			go func() {
				defer GinkgoRecover()
				err := client.Delete(ctx, NetworkName, v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				finished <- true
			}()

			Eventually(events, CTimeout, CInterval).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Deleted))
			eventNetwork = event.Object.(*v1alpha1.Network)
			Expect(eventNetwork).NotTo(BeNil())
			Expect(eventNetwork.Name).To(Equal(NetworkName))

			<-finished

			watcher.Stop()
			Eventually(events, CTimeout, CInterval).Should(BeClosed())
		})
	})
})
