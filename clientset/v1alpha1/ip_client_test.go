package v1alpha1

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/onmetal/ipam/api/v1alpha1"
)

var _ = Describe("Ip client", func() {
	const (
		IpName         = "test-ip"
		IpToDeleteName = "test-ip-to-delete"
		DeleteLabel    = "delete-label"
		IpNamespace    = "default"

		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)
	ipMustParse := func(ipString string) *v1alpha1.IP {
		ip, err := v1alpha1.IPFromString(ipString)
		if err != nil {
			panic(err)
		}
		return ip
	}

	Context("When Ip CR is installed", func() {
		It("Should check that Ip CR is operational with client", func() {
			By("Creating client")
			finished := make(chan bool)
			ctx := context.Background()

			clientset, err := NewForConfig(cfg)
			Expect(err).NotTo(HaveOccurred())

			client := clientset.Ips(IpNamespace)

			ip := &v1alpha1.Ip{
				ObjectMeta: v1.ObjectMeta{
					Name:      IpName,
					Namespace: IpNamespace,
				},
				Spec: v1alpha1.IpSpec{
					Subnet: "sn",
					IP:     ipMustParse("192.168.1.1"),
				},
			}

			By("Creating watcher")
			watcher, err := client.Watch(ctx, v1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			events := watcher.ResultChan()

			By("Creating Ip")
			createdIp := &v1alpha1.Ip{}
			go func() {
				defer GinkgoRecover()
				createdIp, err = client.Create(ctx, ip, v1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(createdIp.Spec).Should(Equal(ip.Spec))
				finished <- true
			}()

			event := &watch.Event{}
			Eventually(events).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Added))
			eventIp := event.Object.(*v1alpha1.Ip)
			Expect(eventIp).NotTo(BeNil())
			Expect(eventIp.Spec).Should(Equal(ip.Spec))

			<-finished

			By("Updating Ip")
			createdIp.Spec.IP = ipMustParse("127.0.0.1")
			updatedIp := &v1alpha1.Ip{}
			go func() {
				defer GinkgoRecover()
				updatedIp, err = client.Update(ctx, createdIp, v1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedIp.Spec).Should(Equal(createdIp.Spec))
				finished <- true
			}()

			Eventually(events).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Modified))
			eventIp = event.Object.(*v1alpha1.Ip)
			Expect(eventIp).NotTo(BeNil())
			Expect(eventIp.Spec).Should(Equal(createdIp.Spec))

			<-finished

			By("Updating Ip status")
			updatedIp.Status.LastUsedIP = ipMustParse("127.0.0.1")
			go func() {
				defer GinkgoRecover()
				statusUpdatedIp, err := client.UpdateStatus(ctx, updatedIp, v1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(statusUpdatedIp.Status).Should(Equal(updatedIp.Status))
				finished <- true
			}()

			Eventually(events).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Modified))
			eventIp = event.Object.(*v1alpha1.Ip)
			Expect(eventIp).NotTo(BeNil())
			Expect(eventIp.Status).Should(Equal(updatedIp.Status))

			<-finished

			By("Patching Ip")
			patch := []struct {
				Op    string `json:"op"`
				Path  string `json:"path"`
				Value string `json:"value"`
			}{{
				Op:    "replace",
				Path:  "/spec/subnet",
				Value: "test-subnet",
			}}

			patchData, err := json.Marshal(patch)
			Expect(err).NotTo(HaveOccurred())

			go func() {
				defer GinkgoRecover()
				patchedIp, err := client.Patch(ctx, IpName, types.JSONPatchType, patchData, v1.PatchOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(patchedIp.Spec.Subnet).Should(Equal(patch[0].Value))
				finished <- true
			}()

			Eventually(events).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Modified))
			eventIp = event.Object.(*v1alpha1.Ip)
			Expect(eventIp).NotTo(BeNil())
			Expect(eventIp.Spec.Subnet).Should(Equal(patch[0].Value))

			<-finished

			ipToDelete := &v1alpha1.Ip{
				ObjectMeta: v1.ObjectMeta{
					Name:      IpToDeleteName,
					Namespace: IpNamespace,
					Labels: map[string]string{
						DeleteLabel: "",
					},
				},
				Spec: v1alpha1.IpSpec{
					Subnet: "sn",
				},
			}

			By("Creating Ip collection")
			_, err = client.Create(ctx, ipToDelete, v1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(events).Should(Receive())

			By("Listing Ips")
			ipList, err := client.List(ctx, v1.ListOptions{})
			Expect(ipList).NotTo(BeNil())
			Expect(ipList.Items).To(HaveLen(2))

			By("Bulk deleting Ip")
			Expect(client.DeleteCollection(ctx, v1.DeleteOptions{}, v1.ListOptions{LabelSelector: DeleteLabel})).To(Succeed())

			By("Requesting created Ip")
			Eventually(func() bool {
				_, err = client.Get(ctx, IpName, v1.GetOptions{})
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Eventually(func() bool {
				_, err = client.Get(ctx, IpToDeleteName, v1.GetOptions{})
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeFalse())

			Eventually(events).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Deleted))
			eventIp = event.Object.(*v1alpha1.Ip)
			Expect(eventIp).NotTo(BeNil())
			Expect(eventIp.Name).To(Equal(IpToDeleteName))

			By("Deleting Ip")
			go func() {
				defer GinkgoRecover()
				err := client.Delete(ctx, IpName, v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				finished <- true
			}()

			Eventually(events).Should(Receive(event))
			Expect(event.Type).To(Equal(watch.Deleted))
			eventIp = event.Object.(*v1alpha1.Ip)
			Expect(eventIp).NotTo(BeNil())
			Expect(eventIp.Name).To(Equal(IpName))

			<-finished

			watcher.Stop()
			Eventually(events).Should(BeClosed())
		})
	})
})
