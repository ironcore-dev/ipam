package v1alpha1

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/onmetal/ipam/api/v1alpha1"
)

var _ = Describe("IP client", func() {
	const (
		IPName         = "test-ip"
		IPToDeleteName = "test-ip-to-delete"
		DeleteLabel    = "delete-label"
		IPNamespace    = "default"

		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)
	ipMustParse := func(ipString string) *v1alpha1.IPAddr {
		ip, err := v1alpha1.IPAddrFromString(ipString)
		if err != nil {
			panic(err)
		}
		return ip
	}

	Context("When IP CR is installed", func() {
		It("Should check that IP CR is operational with client", func() {
			By("Creating client")
			finished := make(chan bool)
			ctx := context.Background()

			clientset, err := NewForConfig(cfg)
			Expect(err).NotTo(HaveOccurred())

			client := clientset.IPs(IPNamespace)

			ip := &v1alpha1.IP{
				ObjectMeta: v1.ObjectMeta{
					Name:      IPName,
					Namespace: IPNamespace,
				},
				Spec: v1alpha1.IPSpec{
					Subnet: corev1.LocalObjectReference{
						Name: "sn",
					},
					IP: ipMustParse("192.168.1.1"),
				},
			}

			By("Creating watcher")
			watcher, err := client.Watch(ctx, v1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			events := watcher.ResultChan()

			By("Creating IP")
			createdIP := &v1alpha1.IP{}
			go func() {
				defer GinkgoRecover()
				createdIP, err = client.Create(ctx, ip, v1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(createdIP.Spec).Should(Equal(ip.Spec))
				finished <- true
			}()

			event := &watch.Event{}
			Eventually(events).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Added))
			eventIP := event.Object.(*v1alpha1.IP)
			Expect(eventIP).NotTo(BeNil())
			Expect(eventIP.Spec).Should(Equal(ip.Spec))

			<-finished

			By("Updating IP")
			createdIP.Spec.IP = ipMustParse("127.0.0.1")
			updatedIP := &v1alpha1.IP{}
			go func() {
				defer GinkgoRecover()
				updatedIP, err = client.Update(ctx, createdIP, v1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedIP.Spec).Should(Equal(createdIP.Spec))
				finished <- true
			}()

			Eventually(events).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Modified))
			eventIP = event.Object.(*v1alpha1.IP)
			Expect(eventIP).NotTo(BeNil())
			Expect(eventIP.Spec).Should(Equal(createdIP.Spec))

			<-finished

			By("Updating IP status")
			updatedIP.Status.Reserved = ipMustParse("127.0.0.1")
			go func() {
				defer GinkgoRecover()
				statusUpdatedIP, err := client.UpdateStatus(ctx, updatedIP, v1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(statusUpdatedIP.Status).Should(Equal(updatedIP.Status))
				finished <- true
			}()

			Eventually(events).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Modified))
			eventIP = event.Object.(*v1alpha1.IP)
			Expect(eventIP).NotTo(BeNil())
			Expect(eventIP.Status).Should(Equal(updatedIP.Status))

			<-finished

			By("Patching IP")
			patch := []struct {
				Op    string `json:"op"`
				Path  string `json:"path"`
				Value string `json:"value"`
			}{{
				Op:    "replace",
				Path:  "/spec/subnet/name",
				Value: "test-subnet",
			}}

			patchData, err := json.Marshal(patch)
			Expect(err).NotTo(HaveOccurred())

			go func() {
				defer GinkgoRecover()
				patchedIP, err := client.Patch(ctx, IPName, types.JSONPatchType, patchData, v1.PatchOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(patchedIP.Spec.Subnet.Name).Should(Equal(patch[0].Value))
				finished <- true
			}()

			Eventually(events).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Modified))
			eventIP = event.Object.(*v1alpha1.IP)
			Expect(eventIP).NotTo(BeNil())
			Expect(eventIP.Spec.Subnet.Name).Should(Equal(patch[0].Value))

			<-finished

			ipToDelete := &v1alpha1.IP{
				ObjectMeta: v1.ObjectMeta{
					Name:      IPToDeleteName,
					Namespace: IPNamespace,
					Labels: map[string]string{
						DeleteLabel: "",
					},
				},
				Spec: v1alpha1.IPSpec{
					Subnet: corev1.LocalObjectReference{
						Name: "sn",
					},
				},
			}

			By("Creating IP collection")
			_, err = client.Create(ctx, ipToDelete, v1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(events).Should(Receive())

			By("Listing IPs")
			ipList, err := client.List(ctx, v1.ListOptions{})
			Expect(ipList).NotTo(BeNil())
			Expect(ipList.Items).To(HaveLen(2))

			By("Bulk deleting IP")
			Expect(client.DeleteCollection(ctx, v1.DeleteOptions{}, v1.ListOptions{LabelSelector: DeleteLabel})).To(Succeed())

			By("Requesting created IP")
			Eventually(func() bool {
				_, err = client.Get(ctx, IPName, v1.GetOptions{})
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Eventually(func() bool {
				_, err = client.Get(ctx, IPToDeleteName, v1.GetOptions{})
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeFalse())

			Eventually(events).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Deleted))
			eventIP = event.Object.(*v1alpha1.IP)
			Expect(eventIP).NotTo(BeNil())
			Expect(eventIP.Name).To(Equal(IPToDeleteName))

			By("Deleting IP")
			go func() {
				defer GinkgoRecover()
				err := client.Delete(ctx, IPName, v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				finished <- true
			}()

			Eventually(events).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Deleted))
			eventIP = event.Object.(*v1alpha1.IP)
			Expect(eventIP).NotTo(BeNil())
			Expect(eventIP.Name).To(Equal(IPName))

			<-finished

			watcher.Stop()
			Eventually(events).Should(BeClosed())
		})
	})
})
