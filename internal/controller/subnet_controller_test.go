// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ironcore-dev/ipam/api/ipam/v1alpha1"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("Subnet controller", func() {
	ns := SetupTest()

	const (
		NetworkName      = "test-network"
		ParentSubnetName = "test-parent-subnet"
		SubnetName       = "test-subnet"
	)

	AfterEach(func(ctx SpecContext) {
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
			Expect(k8sClient.DeleteAllOf(ctx, r.res, client.InNamespace(ns.Name))).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.List(ctx, r.list)
				if err != nil {
					return false
				}
				if r.count(r.list) > 0 {
					return false
				}
				return true
			}).Should(BeTrue())
		}
	})

	It("Should reserve CIDR in parent Network", func(ctx SpecContext) {
		By("Network is installed")
		testNetwork := v1alpha1.Network{
			ObjectMeta: v1.ObjectMeta{
				Name:      NetworkName,
				Namespace: ns.Name,
			},
			Spec: v1alpha1.NetworkSpec{
				Description: "test network",
			},
		}

		Expect(k8sClient.Create(ctx, &testNetwork)).To(Succeed())

		createdNetwork := v1alpha1.Network{}
		testNetworkNamespacedName := types.NamespacedName{
			Namespace: ns.Name,
			Name:      NetworkName,
		}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, testNetworkNamespacedName, &createdNetwork)
			return err == nil
		}).Should(BeTrue())

		By("Subnet is installed")
		testCidr, err := v1alpha1.CIDRFromString("10.0.0.0/8")
		Expect(err).NotTo(HaveOccurred())
		Expect(testCidr).NotTo(BeNil())

		testSubnet := v1alpha1.Subnet{
			ObjectMeta: v1.ObjectMeta{
				Name:      SubnetName,
				Namespace: ns.Name,
			},
			Spec: v1alpha1.SubnetSpec{
				CIDR: testCidr,
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

		Expect(k8sClient.Create(ctx, &testSubnet)).To(Succeed())

		createdSubnet := v1alpha1.Subnet{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: SubnetName}, &createdSubnet)
			if err != nil {
				return false
			}
			if createdSubnet.Status.State == "" {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Subnet has updated status")
		Eventually(Object(&createdSubnet)).Should(SatisfyAll(
			HaveField("Status.Capacity.Value()", testCidr.AddressCapacity().Int64()),
			HaveField("Status.CapacityLeft.Value()", testCidr.AddressCapacity().Int64()),
			HaveField("Status.Locality", v1alpha1.LocalSubnetLocalityType),
			HaveField("Status.Vacant", HaveLen(1)),
			HaveField("Status.Vacant", ContainElement(*testCidr)),
			HaveField("Status.Type", v1alpha1.IPv4SubnetType),
			HaveField("Status.Message", BeEmpty())))

		By("Subnet CIDR is reserved in Network")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: SubnetName}, &createdSubnet)
			if err != nil {
				return false
			}
			if createdSubnet.Status.State != v1alpha1.FinishedSubnetState {
				return false
			}
			return true
		}).Should(BeTrue())

		Expect(k8sClient.Get(ctx, testNetworkNamespacedName, &createdNetwork)).To(Succeed())

		Expect(func() bool {
			for _, cidr := range createdNetwork.Status.IPv4Ranges {
				if cidr.Equal(testCidr) {
					return true
				}
			}
			return false
		}()).To(BeTrue())

		By("Subnet copy is created")
		subnetCopyName := createdSubnet.Name + "-copy"
		subnetCopy := v1alpha1.Subnet{
			ObjectMeta: v1.ObjectMeta{
				Name:      subnetCopyName,
				Namespace: ns.Name,
			},
			Spec: *createdSubnet.Spec.DeepCopy(),
		}
		subnetCopy.Spec.CIDR = testCidr

		Expect(k8sClient.Create(ctx, &subnetCopy)).To(Succeed())

		By("Subnet copy is failed to get CIDR reserved")
		subnetCopyNamespacedName := types.NamespacedName{
			Namespace: ns.Name,
			Name:      subnetCopyName,
		}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, subnetCopyNamespacedName, &subnetCopy)
			if err != nil {
				return false
			}
			if subnetCopy.Status.State != v1alpha1.FailedSubnetState {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Subnet is deleted")
		Expect(k8sClient.Delete(ctx, &createdSubnet)).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: createdSubnet.Name}, &createdSubnet)
			return apierrors.IsNotFound(err)
		}).Should(BeTrue())

		By("Subnet copy gets CIDR reserved")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, subnetCopyNamespacedName, &subnetCopy)
			if err != nil {
				return false
			}
			if subnetCopy.Status.State != v1alpha1.FinishedSubnetState {
				return false
			}
			if !subnetCopy.Status.Reserved.Equal(testCidr) {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Subnet copy is deleted")
		Expect(k8sClient.Delete(ctx, &subnetCopy)).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: subnetCopy.Name}, &subnetCopy)
			return apierrors.IsNotFound(err)
		}).Should(BeTrue())

		By("Subnet CIDR is released in Network")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, testNetworkNamespacedName, &createdNetwork)
			if err != nil {
				return false
			}
			if len(createdNetwork.Status.IPv4Ranges) != 0 {
				return false
			}
			return true
		}).Should(BeTrue())
	})

	It("Should reserve CIDR in parent Subnet", func(ctx SpecContext) {
		By("Network is installed")
		testNetwork := v1alpha1.Network{
			ObjectMeta: v1.ObjectMeta{
				Name:      NetworkName,
				Namespace: ns.Name,
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
				Namespace: ns.Name,
			},
			Spec: v1alpha1.SubnetSpec{
				CIDR: parentSubnetCidr,
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

		Expect(k8sClient.Create(ctx, &testParentSubnet)).To(Succeed())

		createdParentSubnet := v1alpha1.Subnet{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: ParentSubnetName}, &createdParentSubnet)
			if err != nil {
				return false
			}
			if createdParentSubnet.Status.State != v1alpha1.FinishedSubnetState {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Subnet is installed")
		testCidr, err := v1alpha1.CIDRFromString("10.0.1.0/24")
		Expect(err).NotTo(HaveOccurred())
		Expect(testCidr).NotTo(BeNil())

		testSubnet := v1alpha1.Subnet{
			ObjectMeta: v1.ObjectMeta{
				Name:      SubnetName,
				Namespace: ns.Name,
			},
			Spec: v1alpha1.SubnetSpec{
				CIDR: testCidr,
				Network: corev1.LocalObjectReference{
					Name: NetworkName,
				},
				ParentSubnet: corev1.LocalObjectReference{
					Name: ParentSubnetName,
				},
				Regions: []v1alpha1.Region{
					{
						Name:              "euw",
						AvailabilityZones: []string{"a"},
					},
				},
			},
		}

		Expect(k8sClient.Create(ctx, &testSubnet)).To(Succeed())

		createdSubnet := v1alpha1.Subnet{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: SubnetName}, &createdSubnet)
			if err != nil {
				return false
			}
			if createdSubnet.Status.State == "" {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Subnet has updated status")
		Expect(createdSubnet.Status.Capacity.Value()).To(Equal(testCidr.AddressCapacity().Int64()))
		Expect(createdSubnet.Status.CapacityLeft.Value()).To(Equal(testCidr.AddressCapacity().Int64()))
		Expect(createdSubnet.Status.Locality).To(Equal(v1alpha1.LocalSubnetLocalityType))
		Expect(createdSubnet.Status.Vacant).To(HaveLen(1))
		Expect(createdSubnet.Status.Vacant[0].Equal(testCidr)).To(BeTrue())
		Expect(createdSubnet.Status.Type).To(Equal(v1alpha1.IPv4SubnetType))
		Expect(createdSubnet.Status.Message).To(BeZero())

		By("Subnet CIDR is reserved in parent Subnet")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: createdSubnet.Name}, &createdSubnet)
			if err != nil {
				return false
			}
			if createdSubnet.Status.State != v1alpha1.FinishedSubnetState {
				return false
			}
			return true
		}).Should(BeTrue())

		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: createdParentSubnet.Name}, &createdParentSubnet)).To(Succeed())

		Expect(createdParentSubnet.CanReserve(testCidr)).To(BeFalse())
		Expect(createdParentSubnet.CanRelease(testCidr)).To(BeTrue())

		By("Subnet copy is created")
		subnetCopyName := createdSubnet.Name + "-copy"
		subnetCopy := v1alpha1.Subnet{
			ObjectMeta: v1.ObjectMeta{
				Name:      subnetCopyName,
				Namespace: ns.Name,
			},
			Spec: *createdSubnet.Spec.DeepCopy(),
		}
		subnetCopy.Spec.CIDR = testCidr

		Expect(k8sClient.Create(ctx, &subnetCopy)).To(Succeed())

		By("Subnet copy is failed to get CIDR reserved")
		subnetCopyNamespacedName := types.NamespacedName{
			Namespace: ns.Name,
			Name:      subnetCopyName,
		}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, subnetCopyNamespacedName, &subnetCopy)
			if err != nil {
				return false
			}
			if subnetCopy.Status.State != v1alpha1.FailedSubnetState {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Subnet is deleted")
		Expect(k8sClient.Delete(ctx, &createdSubnet)).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: createdSubnet.Name}, &createdSubnet)
			return apierrors.IsNotFound(err)
		}).Should(BeTrue())

		By("Subnet copy gets CIDR reserved")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, subnetCopyNamespacedName, &subnetCopy)
			if err != nil {
				return false
			}
			if subnetCopy.Status.State != v1alpha1.FinishedSubnetState {
				return false
			}
			if !subnetCopy.Status.Reserved.Equal(testCidr) {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Subnet copy is deleted")
		Expect(k8sClient.Delete(ctx, &subnetCopy)).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: subnetCopy.Name}, &subnetCopy)
			return apierrors.IsNotFound(err)
		}).Should(BeTrue())

		By("Subnet CIDR is released in parent Subnet")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: createdParentSubnet.Name}, &createdParentSubnet)
			if err != nil {
				return false
			}
			if !createdParentSubnet.CanReserve(testCidr) {
				return false
			}
			if createdParentSubnet.CanRelease(testCidr) {
				return false
			}
			return true
		}).Should(BeTrue())
	})

	It("Should reserve CIDR in parent Subnet", func(ctx SpecContext) {
		By("Network is installed")
		testNetwork := v1alpha1.Network{
			ObjectMeta: v1.ObjectMeta{
				Name:      NetworkName,
				Namespace: ns.Name,
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
				Namespace: ns.Name,
			},
			Spec: v1alpha1.SubnetSpec{
				CIDR: parentSubnetCidr,
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

		Expect(k8sClient.Create(ctx, &testParentSubnet)).To(Succeed())

		createdParentSubnet := v1alpha1.Subnet{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: ParentSubnetName}, &createdParentSubnet)
			if err != nil {
				return false
			}
			if createdParentSubnet.Status.State != v1alpha1.FinishedSubnetState {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Subnet is installed")
		childSubnetCapacity := int64(256)
		childSubnetCidr, _ := v1alpha1.CIDRFromString("10.0.0.0/24")
		testSubnet := v1alpha1.Subnet{
			ObjectMeta: v1.ObjectMeta{
				Name:      SubnetName,
				Namespace: ns.Name,
			},
			Spec: v1alpha1.SubnetSpec{
				Capacity: resource.NewScaledQuantity(childSubnetCapacity, 0),
				Network: corev1.LocalObjectReference{
					Name: NetworkName,
				},
				ParentSubnet: corev1.LocalObjectReference{
					Name: ParentSubnetName,
				},
				Regions: []v1alpha1.Region{
					{
						Name:              "euw",
						AvailabilityZones: []string{"a"},
					},
				},
			},
		}

		Expect(k8sClient.Create(ctx, &testSubnet)).To(Succeed())

		createdSubnet := v1alpha1.Subnet{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: SubnetName}, &createdSubnet)
			if err != nil {
				return false
			}
			if createdSubnet.Status.State == "" {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Subnet has updated status")
		Expect(createdSubnet.Status.Locality).To(Equal(v1alpha1.LocalSubnetLocalityType))
		Expect(createdSubnet.Status.Message).To(BeZero())

		By("Subnet CIDR is reserved in parent Subnet")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: createdSubnet.Name}, &createdSubnet)
			if err != nil {
				return false
			}
			if createdSubnet.Status.State != v1alpha1.FinishedSubnetState {
				return false
			}
			return true
		}).Should(BeTrue())

		Expect(createdSubnet.Status.Type).To(Equal(v1alpha1.IPv4SubnetType))
		Expect(createdSubnet.Status.Capacity.Value()).To(Equal(childSubnetCapacity))
		Expect(createdSubnet.Status.CapacityLeft.Value()).To(Equal(childSubnetCapacity))
		Expect(createdSubnet.Status.Reserved.Equal(childSubnetCidr)).To(BeTrue())
		Expect(createdSubnet.Status.Vacant).To(HaveLen(1))
		Expect(createdSubnet.Status.Vacant[0].Equal(childSubnetCidr)).To(BeTrue())

		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: createdParentSubnet.Name}, &createdParentSubnet)).To(Succeed())

		Expect(createdParentSubnet.CanReserve(childSubnetCidr)).To(BeFalse())
		Expect(createdParentSubnet.CanRelease(childSubnetCidr)).To(BeTrue())
	})

	It("Should reserve CIDR in parent Subnet", func(ctx SpecContext) {
		By("Network is installed")
		testNetwork := v1alpha1.Network{
			ObjectMeta: v1.ObjectMeta{
				Name:      NetworkName,
				Namespace: ns.Name,
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
				Namespace: ns.Name,
			},
			Spec: v1alpha1.SubnetSpec{
				CIDR: parentSubnetCidr,
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

		Expect(k8sClient.Create(ctx, &testParentSubnet)).To(Succeed())

		createdParentSubnet := v1alpha1.Subnet{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: ParentSubnetName}, &createdParentSubnet)
			if err != nil {
				return false
			}
			if createdParentSubnet.Status.State != v1alpha1.FinishedSubnetState {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Subnet is installed")
		hib := byte(24)
		testSubnet := v1alpha1.Subnet{
			ObjectMeta: v1.ObjectMeta{
				Name:      SubnetName,
				Namespace: ns.Name,
			},
			Spec: v1alpha1.SubnetSpec{
				PrefixBits: &hib,
				Network: corev1.LocalObjectReference{
					Name: NetworkName,
				},
				ParentSubnet: corev1.LocalObjectReference{
					Name: ParentSubnetName,
				},
				Regions: []v1alpha1.Region{
					{
						Name:              "euw",
						AvailabilityZones: []string{"a"},
					},
				},
			},
		}

		Expect(k8sClient.Create(ctx, &testSubnet)).To(Succeed())

		createdSubnet := v1alpha1.Subnet{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: SubnetName}, &createdSubnet)
			if err != nil {
				return false
			}
			if createdSubnet.Status.State == "" {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Subnet has updated status")
		Expect(createdSubnet.Status.Locality).To(Equal(v1alpha1.LocalSubnetLocalityType))
		Expect(createdSubnet.Status.Message).To(BeZero())

		By("Subnet CIDR is reserved in parent Subnet")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: createdSubnet.Name}, &createdSubnet)
			if err != nil {
				return false
			}
			if createdSubnet.Status.State != v1alpha1.FinishedSubnetState {
				return false
			}
			return true
		}).Should(BeTrue())

		maskBits := parentSubnetCidr.MaskBits()
		childSubnetCapacity := int64(1 << (maskBits - hib))
		childSubnetCidr, _ := v1alpha1.CIDRFromString("10.0.0.0/24")
		Expect(createdSubnet.Status.Type).To(Equal(v1alpha1.IPv4SubnetType))
		Expect(createdSubnet.Status.Capacity.Value()).To(Equal(childSubnetCapacity))
		Expect(createdSubnet.Status.CapacityLeft.Value()).To(Equal(childSubnetCapacity))
		Expect(createdSubnet.Status.Reserved.Equal(childSubnetCidr)).To(BeTrue())
		Expect(createdSubnet.Status.Vacant).To(HaveLen(1))
		Expect(createdSubnet.Status.Vacant[0].Equal(childSubnetCidr)).To(BeTrue())

		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: createdParentSubnet.Name}, &createdParentSubnet)).To(Succeed())

		Expect(createdParentSubnet.CanReserve(createdSubnet.Status.Reserved)).To(BeFalse())
		Expect(createdParentSubnet.CanRelease(createdSubnet.Status.Reserved)).To(BeTrue())
	})

	It("Should fall into the failed state", func(ctx SpecContext) {
		By("Network is installed")
		testNetwork := v1alpha1.Network{
			ObjectMeta: v1.ObjectMeta{
				Name:      NetworkName,
				Namespace: ns.Name,
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
				Namespace: ns.Name,
			},
			Spec: v1alpha1.SubnetSpec{
				CIDR: parentSubnetCidr,
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

		Expect(k8sClient.Create(ctx, &testParentSubnet)).To(Succeed())

		createdParentSubnet := v1alpha1.Subnet{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: ParentSubnetName}, &createdParentSubnet)
			if err != nil {
				return false
			}
			if createdParentSubnet.Status.State != v1alpha1.FinishedSubnetState {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Top level Subnet with the same CIDR is installed")
		anotherTopLevelSubnetName := "test-another-top-level-subnet"
		anotherTopLevelSubnet := v1alpha1.Subnet{
			ObjectMeta: v1.ObjectMeta{
				Name:      anotherTopLevelSubnetName,
				Namespace: ns.Name,
			},
			Spec: v1alpha1.SubnetSpec{
				CIDR: parentSubnetCidr,
				Network: corev1.LocalObjectReference{
					Name: NetworkName,
				},
				Regions: []v1alpha1.Region{
					{
						Name:              "eun",
						AvailabilityZones: []string{"b"},
					},
				},
			},
		}

		Expect(k8sClient.Create(ctx, &anotherTopLevelSubnet)).To(Succeed())

		By("Top level Subnet goes into the failed state")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: anotherTopLevelSubnet.Name}, &anotherTopLevelSubnet)
			if err != nil {
				return false
			}
			if anotherTopLevelSubnet.Status.State != v1alpha1.FailedSubnetState {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Subnet is installed")
		testCidr, err := v1alpha1.CIDRFromString("10.0.1.0/24")
		Expect(err).NotTo(HaveOccurred())
		Expect(testCidr).NotTo(BeNil())

		testSubnet := v1alpha1.Subnet{
			ObjectMeta: v1.ObjectMeta{
				Name:      SubnetName,
				Namespace: ns.Name,
			},
			Spec: v1alpha1.SubnetSpec{
				CIDR: testCidr,
				Network: corev1.LocalObjectReference{
					Name: NetworkName,
				},
				ParentSubnet: corev1.LocalObjectReference{
					Name: ParentSubnetName,
				},
				Regions: []v1alpha1.Region{
					{
						Name:              "euw",
						AvailabilityZones: []string{"a"},
					},
				},
			},
		}

		Expect(k8sClient.Create(ctx, &testSubnet)).To(Succeed())

		createdSubnet := v1alpha1.Subnet{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: SubnetName}, &createdSubnet)
			if err != nil {
				return false
			}
			if createdParentSubnet.Status.State != v1alpha1.FinishedSubnetState {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Child Subnet with the same CIDR installed")
		anotherChildSubnetName := "test-another-child-subnet"
		anotherChildSubnet := v1alpha1.Subnet{
			ObjectMeta: v1.ObjectMeta{
				Name:      anotherChildSubnetName,
				Namespace: ns.Name,
			},
			Spec: v1alpha1.SubnetSpec{
				CIDR: testCidr,
				Network: corev1.LocalObjectReference{
					Name: NetworkName,
				},
				ParentSubnet: corev1.LocalObjectReference{
					Name: ParentSubnetName,
				},
				Regions: []v1alpha1.Region{
					{
						Name:              "euw",
						AvailabilityZones: []string{"a"},
					},
				},
			},
		}

		Expect(k8sClient.Create(ctx, &anotherChildSubnet)).To(Succeed())

		By("Child Subnet goes into the failed state")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: anotherChildSubnet.Name}, &anotherChildSubnet)
			if err != nil {
				return false
			}
			if anotherChildSubnet.Status.State != v1alpha1.FailedSubnetState {
				return false
			}
			return true
		}).Should(BeTrue())
	})

	It("Should fall into the failed state", func(ctx SpecContext) {
		By("Network is installed")
		testNetwork := v1alpha1.Network{
			ObjectMeta: v1.ObjectMeta{
				Name:      NetworkName,
				Namespace: ns.Name,
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
				Namespace: ns.Name,
			},
			Spec: v1alpha1.SubnetSpec{
				CIDR: parentSubnetCidr,
				Network: corev1.LocalObjectReference{
					Name: NetworkName,
				},
				Regions: []v1alpha1.Region{
					{
						Name:              "euw",
						AvailabilityZones: []string{"a", "b", "c"},
					},
					{
						Name:              "eun",
						AvailabilityZones: []string{"a", "b", "c"},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, &testParentSubnet)).To(Succeed())

		createdParentSubnet := v1alpha1.Subnet{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: ParentSubnetName}, &createdParentSubnet)
			if err != nil {
				return false
			}
			if createdParentSubnet.Status.State != v1alpha1.FinishedSubnetState {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Child Subnet with wrong regions is installed")
		wrongRegionsChildSubnetName := "wrong-regions-child-subnet"
		wrongRegionsChildSubnet := v1alpha1.Subnet{
			ObjectMeta: v1.ObjectMeta{
				Name:      wrongRegionsChildSubnetName,
				Namespace: ns.Name,
			},
			Spec: v1alpha1.SubnetSpec{
				Capacity: resource.NewScaledQuantity(120, 0),
				Network: corev1.LocalObjectReference{
					Name: NetworkName,
				},
				ParentSubnet: corev1.LocalObjectReference{
					Name: ParentSubnetName,
				},
				Regions: []v1alpha1.Region{
					{
						Name:              "us",
						AvailabilityZones: []string{"a"},
					},
				},
			},
		}

		Expect(k8sClient.Create(ctx, &wrongRegionsChildSubnet)).To(Succeed())

		By("Child Subnet with wrong zones goes into the failed state")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: wrongRegionsChildSubnet.Name}, &wrongRegionsChildSubnet)
			if err != nil {
				return false
			}
			if wrongRegionsChildSubnet.Status.State != v1alpha1.FailedSubnetState {
				return false
			}
			return true
		}).Should(BeTrue())

		By("Child Subnet with wrong availability zones is installed")
		wrongAZsChildSubnetName := "wrong-azs-child-subnet"
		wrongAZsChildSubnet := v1alpha1.Subnet{
			ObjectMeta: v1.ObjectMeta{
				Name:      wrongAZsChildSubnetName,
				Namespace: ns.Name,
			},
			Spec: v1alpha1.SubnetSpec{
				Capacity: resource.NewScaledQuantity(120, 0),
				Network: corev1.LocalObjectReference{
					Name: NetworkName,
				},
				ParentSubnet: corev1.LocalObjectReference{
					Name: ParentSubnetName,
				},
				Regions: []v1alpha1.Region{
					{
						Name:              "euw",
						AvailabilityZones: []string{"b", "f"},
					},
				},
			},
		}

		Expect(k8sClient.Create(ctx, &wrongAZsChildSubnet)).To(Succeed())

		By("Child Subnet with wrong zones goes into the failed state")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: wrongAZsChildSubnet.Name}, &wrongAZsChildSubnet)
			if err != nil {
				return false
			}
			if wrongAZsChildSubnet.Status.State != v1alpha1.FailedSubnetState {
				return false
			}
			return true
		}).Should(BeTrue())
	})
})
