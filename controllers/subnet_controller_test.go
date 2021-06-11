package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/onmetal/ipam/api/v1alpha1"
)

var _ = Describe("Subnet controller", func() {
	const (
		NetworkName      = "test-network"
		ParentSubnetName = "test-parent-subnet"
		SubnetName       = "test-subnet"

		SubnetNamespace = "default"

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
		}

		for _, r := range resources {
			Expect(k8sClient.DeleteAllOf(ctx, r.res, client.InNamespace(SubnetNamespace))).To(Succeed())
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

	Context("When Subnet CR is created", func() {
		It("Should reserve CIDR in parent Network", func() {
			By("Network is installed")
			ctx := context.Background()

			testNetwork := v1alpha1.Network{
				ObjectMeta: v1.ObjectMeta{
					Name:      NetworkName,
					Namespace: SubnetNamespace,
				},
				Spec: v1alpha1.NetworkSpec{
					Description: "test network",
				},
			}

			Expect(k8sClient.Create(ctx, &testNetwork)).To(Succeed())

			createdNetwork := v1alpha1.Network{}
			testNetworkNamespacedName := types.NamespacedName{
				Namespace: SubnetNamespace,
				Name:      NetworkName,
			}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, testNetworkNamespacedName, &createdNetwork)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Subnet is installed")
			testCidr, err := v1alpha1.CIDRFromString("10.0.0.0/8")
			Expect(err).NotTo(HaveOccurred())
			Expect(testCidr).NotTo(BeNil())

			testSubnet := v1alpha1.Subnet{
				ObjectMeta: v1.ObjectMeta{
					Name:      SubnetName,
					Namespace: SubnetNamespace,
				},
				Spec: v1alpha1.SubnetSpec{
					CIDR:              *testCidr,
					NetworkName:       NetworkName,
					Regions:           []string{"euw"},
					AvailabilityZones: []string{"a"},
				},
			}

			Expect(k8sClient.Create(ctx, &testSubnet)).To(Succeed())

			createdSubnet := v1alpha1.Subnet{}
			testSubnetNamespacedName := types.NamespacedName{
				Namespace: SubnetNamespace,
				Name:      SubnetName,
			}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, testSubnetNamespacedName, &createdSubnet)
				if err != nil {
					return false
				}
				if createdSubnet.Status.State == "" {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Subnet has updated status")
			Expect(createdSubnet.Status.Capacity.Value()).To(Equal(testCidr.AddressCapacity().Int64()))
			Expect(createdSubnet.Status.CapacityLeft.Value()).To(Equal(testCidr.AddressCapacity().Int64()))
			Expect(createdSubnet.Status.Locality).To(Equal(v1alpha1.CLocalSubnetLocalityType))
			Expect(createdSubnet.Status.Vacant).To(HaveLen(1))
			Expect(createdSubnet.Status.Vacant[0].Equal(testCidr)).To(BeTrue())
			Expect(createdSubnet.Status.Type).To(Equal(v1alpha1.CIPv4SubnetType))
			Expect(createdSubnet.Status.Message).To(BeZero())

			By("Subnet CIDR is reserved in Network")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, testSubnetNamespacedName, &createdSubnet)
				if err != nil {
					return false
				}
				if createdSubnet.Status.State != v1alpha1.CFinishedSubnetState {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Get(ctx, testNetworkNamespacedName, &createdNetwork)).To(Succeed())

			Expect(func() bool {
				for _, cidr := range createdNetwork.Status.Ranges {
					if cidr.Equal(testCidr) {
						return true
					}
				}
				return false
			}()).To(BeTrue())

			By("Subnet is deleted")
			Expect(k8sClient.Delete(ctx, &createdSubnet)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, testSubnetNamespacedName, &createdSubnet)
				if !apierrors.IsNotFound(err) {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Subnet CIDR is released in Network")
			Expect(k8sClient.Get(ctx, testNetworkNamespacedName, &createdNetwork)).To(Succeed())
			Expect(createdNetwork.Status.Ranges).To(HaveLen(0))
		})
	})

	Context("When Subnet CR is created", func() {
		It("Should reserve CIDR in parent Subnet", func() {
			By("Network is installed")
			ctx := context.Background()

			testNetwork := v1alpha1.Network{
				ObjectMeta: v1.ObjectMeta{
					Name:      NetworkName,
					Namespace: SubnetNamespace,
				},
				Spec: v1alpha1.NetworkSpec{
					Description: "test network",
				},
			}

			Expect(k8sClient.Create(ctx, &testNetwork)).To(Succeed())

			By("Parent subnet is installed")
			parentSubnetCidr, err := v1alpha1.CIDRFromString("10.0.0.0/8")
			Expect(err).NotTo(HaveOccurred())
			Expect(parentSubnetCidr).NotTo(BeNil())

			testParentSubnet := v1alpha1.Subnet{
				ObjectMeta: v1.ObjectMeta{
					Name:      ParentSubnetName,
					Namespace: SubnetNamespace,
				},
				Spec: v1alpha1.SubnetSpec{
					CIDR:              *parentSubnetCidr,
					NetworkName:       NetworkName,
					Regions:           []string{"euw"},
					AvailabilityZones: []string{"a"},
				},
			}

			Expect(k8sClient.Create(ctx, &testParentSubnet)).To(Succeed())

			createdParentSubnet := v1alpha1.Subnet{}
			testParentSubnetNamespacedName := types.NamespacedName{
				Namespace: SubnetNamespace,
				Name:      ParentSubnetName,
			}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, testParentSubnetNamespacedName, &createdParentSubnet)
				if err != nil {
					return false
				}
				if createdParentSubnet.Status.State != v1alpha1.CFinishedSubnetState {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Subnet is installed")
			testCidr, err := v1alpha1.CIDRFromString("10.0.1.0/24")
			Expect(err).NotTo(HaveOccurred())
			Expect(testCidr).NotTo(BeNil())

			testSubnet := v1alpha1.Subnet{
				ObjectMeta: v1.ObjectMeta{
					Name:      SubnetName,
					Namespace: SubnetNamespace,
				},
				Spec: v1alpha1.SubnetSpec{
					CIDR:              *testCidr,
					NetworkName:       NetworkName,
					ParentSubnetName:  ParentSubnetName,
					Regions:           []string{"euw"},
					AvailabilityZones: []string{"a"},
				},
			}

			Expect(k8sClient.Create(ctx, &testSubnet)).To(Succeed())

			createdSubnet := v1alpha1.Subnet{}
			testSubnetNamespacedName := types.NamespacedName{
				Namespace: SubnetNamespace,
				Name:      SubnetName,
			}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, testSubnetNamespacedName, &createdSubnet)
				if err != nil {
					return false
				}
				if createdSubnet.Status.State == "" {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Subnet has updated status")
			Expect(createdSubnet.Status.Capacity.Value()).To(Equal(testCidr.AddressCapacity().Int64()))
			Expect(createdSubnet.Status.CapacityLeft.Value()).To(Equal(testCidr.AddressCapacity().Int64()))
			Expect(createdSubnet.Status.Locality).To(Equal(v1alpha1.CLocalSubnetLocalityType))
			Expect(createdSubnet.Status.Vacant).To(HaveLen(1))
			Expect(createdSubnet.Status.Vacant[0].Equal(testCidr)).To(BeTrue())
			Expect(createdSubnet.Status.Type).To(Equal(v1alpha1.CIPv4SubnetType))
			Expect(createdSubnet.Status.Message).To(BeZero())

			By("Subnet CIDR is reserved in parent Subnet")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, testSubnetNamespacedName, &createdSubnet)
				if err != nil {
					return false
				}
				if createdSubnet.Status.State != v1alpha1.CFinishedSubnetState {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Get(ctx, testParentSubnetNamespacedName, &createdParentSubnet)).To(Succeed())

			Expect(createdParentSubnet.CanReserve(testCidr)).To(BeFalse())
			Expect(createdParentSubnet.CanRelease(testCidr)).To(BeTrue())

			By("Subnet is deleted")
			Expect(k8sClient.Delete(ctx, &createdSubnet)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, testSubnetNamespacedName, &createdSubnet)
				if !apierrors.IsNotFound(err) {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Subnet CIDR is released in parent Subnet")
			Expect(k8sClient.Get(ctx, testParentSubnetNamespacedName, &createdParentSubnet)).To(Succeed())
			Expect(createdParentSubnet.CanReserve(testCidr)).To(BeTrue())
			Expect(createdParentSubnet.CanRelease(testCidr)).To(BeFalse())
		})
	})

	Context("When Subnet CR is created with already booked CIDR", func() {
		It("Should fall into the failed state", func() {
			By("Network is installed")
			ctx := context.Background()

			testNetwork := v1alpha1.Network{
				ObjectMeta: v1.ObjectMeta{
					Name:      NetworkName,
					Namespace: SubnetNamespace,
				},
				Spec: v1alpha1.NetworkSpec{
					Description: "test network",
				},
			}

			Expect(k8sClient.Create(ctx, &testNetwork)).To(Succeed())

			By("Parent subnet is installed")
			parentSubnetCidr, err := v1alpha1.CIDRFromString("10.0.0.0/8")
			Expect(err).NotTo(HaveOccurred())
			Expect(parentSubnetCidr).NotTo(BeNil())

			testParentSubnet := v1alpha1.Subnet{
				ObjectMeta: v1.ObjectMeta{
					Name:      ParentSubnetName,
					Namespace: SubnetNamespace,
				},
				Spec: v1alpha1.SubnetSpec{
					CIDR:              *parentSubnetCidr,
					NetworkName:       NetworkName,
					Regions:           []string{"euw"},
					AvailabilityZones: []string{"a"},
				},
			}

			Expect(k8sClient.Create(ctx, &testParentSubnet)).To(Succeed())

			createdParentSubnet := v1alpha1.Subnet{}
			testParentSubnetNamespacedName := types.NamespacedName{
				Namespace: SubnetNamespace,
				Name:      ParentSubnetName,
			}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, testParentSubnetNamespacedName, &createdParentSubnet)
				if err != nil {
					return false
				}
				if createdParentSubnet.Status.State != v1alpha1.CFinishedSubnetState {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Top level Subnet with the same CIDR is installed")
			anotherTopLevelSubnetName := "test-another-top-level-subnet"
			anotherTopLevelSubnetNamespacedName := types.NamespacedName{
				Name:      anotherTopLevelSubnetName,
				Namespace: SubnetNamespace,
			}
			anotherTopLevelSubnet := v1alpha1.Subnet{
				ObjectMeta: v1.ObjectMeta{
					Name:      anotherTopLevelSubnetName,
					Namespace: SubnetNamespace,
				},
				Spec: v1alpha1.SubnetSpec{
					CIDR:              *parentSubnetCidr,
					NetworkName:       NetworkName,
					Regions:           []string{"eun"},
					AvailabilityZones: []string{"b"},
				},
			}

			Expect(k8sClient.Create(ctx, &anotherTopLevelSubnet)).To(Succeed())

			By("Top level Subnet goes into the failed state")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, anotherTopLevelSubnetNamespacedName, &anotherTopLevelSubnet)
				if err != nil {
					return false
				}
				if anotherTopLevelSubnet.Status.State != v1alpha1.CFailedSubnetState {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Subnet is installed")
			testCidr, err := v1alpha1.CIDRFromString("10.0.1.0/24")
			Expect(err).NotTo(HaveOccurred())
			Expect(testCidr).NotTo(BeNil())

			testSubnet := v1alpha1.Subnet{
				ObjectMeta: v1.ObjectMeta{
					Name:      SubnetName,
					Namespace: SubnetNamespace,
				},
				Spec: v1alpha1.SubnetSpec{
					CIDR:              *testCidr,
					NetworkName:       NetworkName,
					ParentSubnetName:  ParentSubnetName,
					Regions:           []string{"euw"},
					AvailabilityZones: []string{"a"},
				},
			}

			Expect(k8sClient.Create(ctx, &testSubnet)).To(Succeed())

			createdSubnet := v1alpha1.Subnet{}
			testSubnetNamespacedName := types.NamespacedName{
				Namespace: SubnetNamespace,
				Name:      SubnetName,
			}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, testSubnetNamespacedName, &createdSubnet)
				if err != nil {
					return false
				}
				if createdParentSubnet.Status.State != v1alpha1.CFinishedSubnetState {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Child Subnet with the same CIDR installed")
			anotherChildSubnetName := "test-another-child-subnet"
			anotherChildSubnetNamespacedName := types.NamespacedName{
				Name:      anotherChildSubnetName,
				Namespace: SubnetNamespace,
			}
			anotherChildSubnet := v1alpha1.Subnet{
				ObjectMeta: v1.ObjectMeta{
					Name:      anotherChildSubnetName,
					Namespace: SubnetNamespace,
				},
				Spec: v1alpha1.SubnetSpec{
					CIDR:              *testCidr,
					NetworkName:       NetworkName,
					ParentSubnetName:  ParentSubnetName,
					Regions:           []string{"euw"},
					AvailabilityZones: []string{"a"},
				},
			}

			Expect(k8sClient.Create(ctx, &anotherChildSubnet)).To(Succeed())

			By("Child Subnet goes into the failed state")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, anotherChildSubnetNamespacedName, &anotherChildSubnet)
				if err != nil {
					return false
				}
				if anotherChildSubnet.Status.State != v1alpha1.CFailedSubnetState {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
		})
	})
})
