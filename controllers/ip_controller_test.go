package controllers

import (
	"context"
	"github.com/onmetal/ipam/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IP controller", func() {
	const (
		timeout  = time.Second * 5
		interval = time.Millisecond * 250

		Namespace   = "default"
		NetworkName = "test-network"
		SubnetName  = "test-subent"
		IPName      = "test-ip"
	)

	cidrMustParse := func(cidrString string) *v1alpha1.CIDR {
		cidr, err := v1alpha1.CIDRFromString(cidrString)
		if err != nil {
			panic(err)
		}
		return cidr
	}

	AfterEach(func() {
		ctx := context.Background()
		resources := []struct {
			res   client.Object
			list  client.ObjectList
			count func(client.ObjectList) int
		}{
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
				res:  &v1alpha1.Ip{},
				list: &v1alpha1.IpList{},
				count: func(objList client.ObjectList) int {
					list := objList.(*v1alpha1.IpList)
					return len(list.Items)
				},
			},
		}

		for _, r := range resources {
			Expect(k8sClient.DeleteAllOf(ctx, r.res, client.InNamespace(Namespace))).To(Succeed())
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

	Context("IP controller test", func() {
		It("Should get IP assigned", func() {
			By("Network is created")
			ctx := context.Background()

			network := &v1alpha1.Network{
				ObjectMeta: metav1.ObjectMeta{
					Name:      NetworkName,
					Namespace: Namespace,
				},
				Spec: v1alpha1.NetworkSpec{},
			}

			Expect(k8sClient.Create(ctx, network)).Should(Succeed())

			networkNamespacedName := types.NamespacedName{
				Name:      NetworkName,
				Namespace: Namespace,
			}
			createdNetwork := &v1alpha1.Network{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, networkNamespacedName, createdNetwork)
				if err != nil {
					return false
				}
				if createdNetwork.Status.State != v1alpha1.CFinishedRequestState {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Subnet is created")
			subnet := &v1alpha1.Subnet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SubnetName,
					Namespace: Namespace,
				},
				Spec: v1alpha1.SubnetSpec{
					CIDR:              cidrMustParse("10.0.0.0/30"),
					NetworkName:       NetworkName,
					Regions:           []string{"euw"},
					AvailabilityZones: []string{"a"},
				},
			}

			Expect(k8sClient.Create(ctx, subnet)).Should(Succeed())

			namespacedName := types.NamespacedName{
				Name:      SubnetName,
				Namespace: Namespace,
			}
			createdSubnet := &v1alpha1.Subnet{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, createdSubnet)
				if err != nil {
					return false
				}
				if createdSubnet.Status.State != v1alpha1.CFinishedSubnetState {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("IP created successfully")
			ip := &v1alpha1.Ip{
				ObjectMeta: metav1.ObjectMeta{
					Name:      IPName,
					Namespace: Namespace,
				},
				Spec: v1alpha1.IpSpec{
					Subnet: SubnetName,
					IP:     "10.0.0.1",
				},
			}

			Expect(k8sClient.Create(ctx, ip)).Should(Succeed())

			By("IP reserved in subnet")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, createdSubnet)
				if err != nil {
					return false
				}
				return len(createdSubnet.Status.Vacant) == 2
			}, timeout, interval).Should(BeTrue())

			By("IP is deleted")
			Expect(k8sClient.Delete(ctx, ip)).Should(Succeed())

			By("IP is released in subnet")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, createdSubnet)
				if err != nil {
					return false
				}
				return len(createdSubnet.Status.Vacant) == 1
			}, timeout, interval).Should(BeTrue())
		})
	})
})
