package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/onmetal/ipam/api/v1alpha1"
)

var _ = Describe("NetworkCounter controller", func() {
	const (
		NetworkCounterNamespace = "default"
		NetworkCounterName      = CVXLANCounterName

		NetworkName = "test-network"

		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)

	AfterEach(func() {
		ctx := context.Background()
		resources := []struct {
			res   client.Object
			list  client.ObjectList
			count func(client.ObjectList) int
		}{
			{
				res:  &v1alpha1.IP{},
				list: &v1alpha1.IPList{},
				count: func(objList client.ObjectList) int {
					list := objList.(*v1alpha1.IPList)
					return len(list.Items)
				},
			},
			{
				res:  &v1alpha1.Subnet{},
				list: &v1alpha1.SubnetList{},
				count: func(objList client.ObjectList) int {
					list := objList.(*v1alpha1.SubnetList)
					return len(list.Items)
				},
			},
			{
				res:  &v1alpha1.Network{},
				list: &v1alpha1.NetworkList{},
				count: func(objList client.ObjectList) int {
					list := objList.(*v1alpha1.NetworkList)
					return len(list.Items)
				},
			},
			{
				res:  &v1alpha1.NetworkCounter{},
				list: &v1alpha1.NetworkCounterList{},
				count: func(objList client.ObjectList) int {
					list := objList.(*v1alpha1.NetworkCounterList)
					return len(list.Items)
				},
			},
		}

		for _, r := range resources {
			Expect(k8sClient.DeleteAllOf(ctx, r.res, client.InNamespace(NetworkCounterNamespace))).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.List(ctx, r.list)
				if err != nil {
					return false
				}
				if r.count(r.list) > 0 {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
		}
	})

	Context("When network counter is updated", func() {
		It("Should trigger update of failed networks", func() {
			By("Counter is created")
			ctx := context.Background()

			counterSpec := v1alpha1.NewNetworkCounterSpec(v1alpha1.CVXLANNetworkType)
			counterSpec.Vacant[0].Begin = v1alpha1.NetworkIDFromInt64(101)

			networkCounter := v1alpha1.NetworkCounter{
				ObjectMeta: metav1.ObjectMeta{
					Name:      NetworkCounterName,
					Namespace: NetworkCounterNamespace,
				},
				Spec: *counterSpec,
			}

			Expect(k8sClient.Create(ctx, &networkCounter)).Should(Succeed())

			By("Network is created")
			network := v1alpha1.Network{
				ObjectMeta: metav1.ObjectMeta{
					Name:      NetworkName,
					Namespace: NetworkCounterNamespace,
				},
				Spec: v1alpha1.NetworkSpec{
					Type: v1alpha1.CVXLANNetworkType,
					ID:   v1alpha1.NetworkIDFromInt64(100),
				},
			}

			Expect(k8sClient.Create(ctx, &network)).Should(Succeed())

			By("Network has failed state")
			Eventually(func() bool {
				networkNamespacedName := types.NamespacedName{
					Namespace: NetworkCounterNamespace,
					Name:      NetworkName,
				}
				updatedNetwork := &v1alpha1.Network{}
				err := k8sClient.Get(ctx, networkNamespacedName, updatedNetwork)
				if err != nil {
					return false
				}
				if updatedNetwork.Status.State != v1alpha1.CFailedNetworkState {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Counter is updated")
			networkCounterNamespacedName := types.NamespacedName{
				Namespace: NetworkCounterNamespace,
				Name:      NetworkCounterName,
			}
			updatedNetworkCounter := &v1alpha1.NetworkCounter{}
			Expect(k8sClient.Get(ctx, networkCounterNamespacedName, updatedNetworkCounter)).Should(Succeed())

			updatedNetworkCounter.Spec.Vacant[0].Begin = v1alpha1.NetworkIDFromInt64(100)
			Expect(k8sClient.Update(ctx, updatedNetworkCounter)).Should(Succeed())

			By("Network has ID assigned")
			Eventually(func() bool {
				networkNamespacedName := types.NamespacedName{
					Namespace: NetworkCounterNamespace,
					Name:      NetworkName,
				}
				updatedNetwork := &v1alpha1.Network{}
				err := k8sClient.Get(ctx, networkNamespacedName, updatedNetwork)
				if err != nil {
					return false
				}
				if updatedNetwork.Status.State != v1alpha1.CFinishedNetworkState {
					return false
				}
				if updatedNetwork.Status.Reserved == nil {
					return false
				}
				if updatedNetwork.Status.Reserved.Int64() != 100 {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
		})
	})
})
