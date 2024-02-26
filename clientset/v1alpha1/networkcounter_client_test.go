// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/ironcore-dev/ipam/api/ipam/v1alpha1"
)

var _ = Describe("NetworkCounter client", func() {
	const (
		NetworkCounterName         = "test-networkcounter"
		NetworkCounterToDeleteName = "test-networkcounter-to-delete"
		DeleteLabel                = "delete-label"
		NetworkCounterNamespace    = "default"
	)

	Context("When NetworkCounter CR is installed", func() {
		It("Should check that NetworkCounter CR is operational with client", func() {
			By("Creating client")
			finished := make(chan bool)

			clientset, err := NewForConfig(cfg)
			Expect(err).NotTo(HaveOccurred())

			client := clientset.NetworkCounters(NetworkCounterNamespace)

			networkCounter := &v1alpha1.NetworkCounter{
				ObjectMeta: v1.ObjectMeta{
					Name:      NetworkCounterName,
					Namespace: NetworkCounterNamespace,
				},
				Spec: v1alpha1.NetworkCounterSpec{
					Vacant: []v1alpha1.NetworkIDInterval{
						{
							Begin: v1alpha1.NetworkIDFromInt64(1),
							End:   v1alpha1.NetworkIDFromInt64(4),
						},
					},
				},
			}

			By("Creating watcher")
			watcher, err := client.Watch(ctx, v1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			events := watcher.ResultChan()

			By("Creating NetworkCounter")
			createdNetworkCounter := &v1alpha1.NetworkCounter{}
			go func() {
				defer GinkgoRecover()
				createdNetworkCounter, err = client.Create(ctx, networkCounter, v1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(createdNetworkCounter.Spec).Should(Equal(networkCounter.Spec))
				finished <- true
			}()

			event := &watch.Event{}
			Eventually(events, CTimeout, CInterval).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Added))
			eventNetworkCounter := event.Object.(*v1alpha1.NetworkCounter)
			Expect(eventNetworkCounter).NotTo(BeNil())
			Expect(eventNetworkCounter.Spec).Should(Equal(networkCounter.Spec))

			<-finished

			By("Updating NetworkCounter")
			createdNetworkCounter.Spec.Vacant[0].End = v1alpha1.NetworkIDFromInt64(5)
			go func() {
				defer GinkgoRecover()
				updatedNetworkCounter, err := client.Update(ctx, createdNetworkCounter, v1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedNetworkCounter.Spec).Should(Equal(createdNetworkCounter.Spec))
				finished <- true
			}()

			Eventually(events, CTimeout, CInterval).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Modified))
			eventNetworkCounter = event.Object.(*v1alpha1.NetworkCounter)
			Expect(eventNetworkCounter).NotTo(BeNil())
			Expect(eventNetworkCounter.Spec).Should(Equal(createdNetworkCounter.Spec))

			<-finished

			By("Updating NetworkCounter status")
			_, err = client.UpdateStatus(ctx, eventNetworkCounter, v1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(events, CTimeout, CInterval).Should(Receive())

			By("Patching NetworkCounter")
			patch := []struct {
				Op    string `json:"op"`
				Path  string `json:"path"`
				Value string `json:"value"`
			}{{
				Op:    "replace",
				Path:  "/spec/vacant/0/begin",
				Value: "2",
			}}

			patchData, err := json.Marshal(patch)
			Expect(err).NotTo(HaveOccurred())

			go func() {
				defer GinkgoRecover()
				patchedNetworkCounter, err := client.Patch(ctx, NetworkCounterName, types.JSONPatchType, patchData, v1.PatchOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(patchedNetworkCounter.Spec.Vacant[0].Begin.String()).Should(Equal(patch[0].Value))
				finished <- true
			}()

			Eventually(events, CTimeout, CInterval).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Modified))
			eventNetworkCounter = event.Object.(*v1alpha1.NetworkCounter)
			Expect(eventNetworkCounter).NotTo(BeNil())
			Expect(eventNetworkCounter.Spec.Vacant[0].Begin.String()).Should(Equal(patch[0].Value))

			<-finished

			networkCounterToDelete := &v1alpha1.NetworkCounter{
				ObjectMeta: v1.ObjectMeta{
					Name:      NetworkCounterToDeleteName,
					Namespace: NetworkCounterNamespace,
					Labels: map[string]string{
						DeleteLabel: "",
					},
				},
				Spec: v1alpha1.NetworkCounterSpec{
					Vacant: []v1alpha1.NetworkIDInterval{
						{
							Exact: v1alpha1.NetworkIDFromInt64(4),
						},
					},
				},
			}

			By("Creating NetworkCounter collection")
			_, err = client.Create(ctx, networkCounterToDelete, v1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(events, CTimeout, CInterval).Should(Receive())

			By("Listing NetworkCounters")
			networkCounterList, err := client.List(ctx, v1.ListOptions{})
			Expect(networkCounterList).NotTo(BeNil())
			Expect(networkCounterList.Items).To(HaveLen(2))

			By("Bulk deleting NetworkCounter")
			Expect(client.DeleteCollection(ctx, v1.DeleteOptions{}, v1.ListOptions{LabelSelector: DeleteLabel})).To(Succeed())

			By("Requesting created NetworkCounter")
			Eventually(func() bool {
				_, err = client.Get(ctx, NetworkCounterName, v1.GetOptions{})
				return err == nil
			}, CTimeout, CInterval).Should(BeTrue())
			Eventually(func() bool {
				_, err = client.Get(ctx, NetworkCounterToDeleteName, v1.GetOptions{})
				return err == nil
			}, CTimeout, CInterval).Should(BeFalse())

			Eventually(events, CTimeout, CInterval).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Deleted))
			eventNetworkCounter = event.Object.(*v1alpha1.NetworkCounter)
			Expect(eventNetworkCounter).NotTo(BeNil())
			Expect(eventNetworkCounter.Name).To(Equal(NetworkCounterToDeleteName))

			By("Deleting NetworkCounter")
			go func() {
				defer GinkgoRecover()
				err := client.Delete(ctx, NetworkCounterName, v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				finished <- true
			}()

			Eventually(events, CTimeout, CInterval).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Deleted))
			eventNetworkCounter = event.Object.(*v1alpha1.NetworkCounter)
			Expect(eventNetworkCounter).NotTo(BeNil())
			Expect(eventNetworkCounter.Name).To(Equal(NetworkCounterName))

			<-finished

			watcher.Stop()
			Eventually(events, CTimeout, CInterval).Should(BeClosed())
		})
	})
})
