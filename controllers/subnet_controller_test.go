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
		NetworkGlobalName = "test-network-global"
		ParentSubnetName  = "test-parent-subnet"
		SubnetName        = "test-subnet"

		SubnetNamespace = "default"

		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)

	AfterEach(func() {
		ctx := context.Background()
		Expect(k8sClient.DeleteAllOf(ctx, &v1alpha1.Subnet{}, client.InNamespace(SubnetNamespace))).To(Succeed())
		Eventually(func() bool {
			list := v1alpha1.SubnetList{}
			err := k8sClient.List(ctx, &list)
			if err != nil {
				return false
			}
			if len(list.Items) > 0 {
				return false
			}
			return true
		}, timeout, interval).Should(BeTrue())
		Expect(k8sClient.DeleteAllOf(ctx, &v1alpha1.NetworkGlobal{}, client.InNamespace(SubnetNamespace))).To(Succeed())
		Eventually(func() bool {
			list := v1alpha1.NetworkGlobalList{}
			err := k8sClient.List(ctx, &list)
			if err != nil {
				return false
			}
			if len(list.Items) > 0 {
				return false
			}
			return true
		}, timeout, interval).Should(BeTrue())
	})

	Context("When Subnet CR is created", func() {
		It("Should reserve CIDR in parent NetworkGlobal", func() {
			By("NetworkGlobal is installed")
			ctx := context.Background()

			testNetworkGlobal := v1alpha1.NetworkGlobal{
				ObjectMeta: v1.ObjectMeta{
					Name:      NetworkGlobalName,
					Namespace: SubnetNamespace,
				},
				Spec: v1alpha1.NetworkGlobalSpec{
					Description: "test network global",
				},
			}

			Expect(k8sClient.Create(ctx, &testNetworkGlobal)).To(Succeed())

			createdNetworkGlobal := v1alpha1.NetworkGlobal{}
			testNetworkGlobalNamespacedName := types.NamespacedName{
				Namespace: SubnetNamespace,
				Name:      NetworkGlobalName,
			}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, testNetworkGlobalNamespacedName, &createdNetworkGlobal)
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
					NetworkGlobalName: NetworkGlobalName,
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

			By("Subnet CIDR is reserved in Network global")
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

			Expect(k8sClient.Get(ctx, testNetworkGlobalNamespacedName, &createdNetworkGlobal)).To(Succeed())

			Expect(func() bool {
				for _, cidr := range createdNetworkGlobal.Status.Ranges {
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

			By("Subnet CIDR is released in Network global")
			Expect(k8sClient.Get(ctx, testNetworkGlobalNamespacedName, &createdNetworkGlobal)).To(Succeed())
			Expect(createdNetworkGlobal.Status.Ranges).To(HaveLen(0))
		})
	})

	Context("When Subnet CR is created", func() {
		It("Should reserve CIDR in parent Subnet", func() {
			By("NetworkGlobal is installed")
			ctx := context.Background()

			testNetworkGlobal := v1alpha1.NetworkGlobal{
				ObjectMeta: v1.ObjectMeta{
					Name:      NetworkGlobalName,
					Namespace: SubnetNamespace,
				},
				Spec: v1alpha1.NetworkGlobalSpec{
					Description: "test network global",
				},
			}

			Expect(k8sClient.Create(ctx, &testNetworkGlobal)).To(Succeed())

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
					NetworkGlobalName: NetworkGlobalName,
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
					NetworkGlobalName: NetworkGlobalName,
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
			By("NetworkGlobal is installed")
			ctx := context.Background()

			testNetworkGlobal := v1alpha1.NetworkGlobal{
				ObjectMeta: v1.ObjectMeta{
					Name:      NetworkGlobalName,
					Namespace: SubnetNamespace,
				},
				Spec: v1alpha1.NetworkGlobalSpec{
					Description: "test network global",
				},
			}

			Expect(k8sClient.Create(ctx, &testNetworkGlobal)).To(Succeed())

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
					NetworkGlobalName: NetworkGlobalName,
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
					NetworkGlobalName: NetworkGlobalName,
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
					NetworkGlobalName: NetworkGlobalName,
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
					NetworkGlobalName: NetworkGlobalName,
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
