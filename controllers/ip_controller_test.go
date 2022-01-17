package controllers

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/onmetal/ipam/api/v1alpha1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IP controller", func() {
	const (
		timeout  = time.Second * 5
		interval = time.Millisecond * 250

		Namespace   = "default"
		NetworkName = "test-network"
		SubnetName  = "test-subnet"
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
				res:  &v1alpha1.IP{},
				list: &v1alpha1.IPList{},
				count: func(objList client.ObjectList) int {
					list := objList.(*v1alpha1.IPList)
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
				if createdNetwork.Status.State != v1alpha1.CFinishedNetworkState {
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
					CIDR: cidrMustParse("10.0.0.0/30"),
					Network: corev1.LocalObjectReference{
						Name: NetworkName,
					},
					Regions: []v1alpha1.Region{
						{
							Name:              "euw",
							AvailabilityZones: []string{"a"},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, subnet)).Should(Succeed())

			subnetNamespacedName := types.NamespacedName{
				Name:      SubnetName,
				Namespace: Namespace,
			}
			createdSubnet := &v1alpha1.Subnet{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, subnetNamespacedName, createdSubnet)
				if err != nil {
					return false
				}
				if createdSubnet.Status.State != v1alpha1.CFinishedSubnetState {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("IP created successfully")
			testIP, err := v1alpha1.IPAddrFromString("10.0.0.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(testIP).NotTo(BeNil())

			ip := &v1alpha1.IP{
				ObjectMeta: metav1.ObjectMeta{
					Name:      IPName,
					Namespace: Namespace,
				},
				Spec: v1alpha1.IPSpec{
					Subnet: corev1.LocalObjectReference{
						Name: SubnetName,
					},
					IP: testIP,
				},
			}

			Expect(k8sClient.Create(ctx, ip)).Should(Succeed())

			ipNamespacedName := types.NamespacedName{
				Name:      IPName,
				Namespace: Namespace,
			}
			createdIP := &v1alpha1.IP{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, ipNamespacedName, createdIP)
				if err != nil {
					return false
				}
				if createdIP.Status.State != v1alpha1.CFinishedIPState {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("IP reserved in subnet")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, subnetNamespacedName, createdSubnet)
				if err != nil {
					return false
				}
				return len(createdSubnet.Status.Vacant) == 2
			}, timeout, interval).Should(BeTrue())

			By("IP is deleted")
			Expect(k8sClient.Delete(ctx, ip)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, ipNamespacedName, createdIP)
				if !apierrors.IsNotFound(err)  {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("IP is released in subnet")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, subnetNamespacedName, createdSubnet)
				if err != nil {
					return false
				}
				return len(createdSubnet.Status.Vacant) == 1
			}, timeout, interval).Should(BeTrue())
		})
	})
})
