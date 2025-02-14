// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"fmt"
	"math"

	"github.com/ironcore-dev/ipam/api/ipam/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Subnet webhook", func() {
	Context("When Network is not created", func() {
		It("Should check that invalid CR will be rejected", func() {
			testNamespaceName := createTestNamespace()

			crs := []v1alpha1.Subnet{
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "without-rules",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha1.SubnetSpec{
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []v1alpha1.Region{
							{
								Name:              "euw",
								AvailabilityZones: []string{"a"},
							},
						},
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "with-more-than-1-rule",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha1.SubnetSpec{
						CIDR:     v1alpha1.CidrMustParse("127.0.0.0/24"),
						Capacity: resource.NewScaledQuantity(60, 0),
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []v1alpha1.Region{
							{
								Name:              "euw",
								AvailabilityZones: []string{"a"},
							},
						},
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "without-parent-subnet-and-with-cidr",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha1.SubnetSpec{
						Capacity: resource.NewScaledQuantity(60, 0),
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []v1alpha1.Region{
							{
								Name:              "euw",
								AvailabilityZones: []string{"a"},
							},
						},
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "with-small-quantity",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha1.SubnetSpec{
						Capacity: resource.NewScaledQuantity(0, 0),
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []v1alpha1.Region{
							{
								Name:              "euw",
								AvailabilityZones: []string{"a"},
							},
						},
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "with-big-quantity",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha1.SubnetSpec{
						Capacity: resource.NewScaledQuantity(math.MaxInt64, resource.Exa),
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []v1alpha1.Region{
							{
								Name:              "euw",
								AvailabilityZones: []string{"a"},
							},
						},
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "with-duplicate-region",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha1.SubnetSpec{
						CIDR: v1alpha1.CidrMustParse("127.0.0.0/24"),
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []v1alpha1.Region{
							{
								Name:              "euw",
								AvailabilityZones: []string{"a"},
							},
							{
								Name:              "euw",
								AvailabilityZones: []string{"a"},
							},
						},
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "with-duplicate-az",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha1.SubnetSpec{
						CIDR: v1alpha1.CidrMustParse("127.0.0.0/24"),
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []v1alpha1.Region{
							{
								Name:              "euw",
								AvailabilityZones: []string{"a", "a"},
							},
							{
								Name:              "eun",
								AvailabilityZones: []string{"a"},
							},
						},
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "with-invalid-consumer-ref",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha1.SubnetSpec{
						CIDR: v1alpha1.CidrMustParse("127.0.0.0/24"),
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []v1alpha1.Region{
							{
								Name:              "euw",
								AvailabilityZones: []string{"a"},
							},
						},
						Consumer: &v1alpha1.ResourceReference{
							APIVersion: "",
							Kind:       "",
						},
					},
				},
			}

			ctx := context.Background()

			for _, cr := range crs {
				By(fmt.Sprintf("Attempting to create Subnet with invalid configuration %s", cr.Name))
				Expect(k8sClient.Create(ctx, &cr)).ShouldNot(Succeed())
			}
		})
	})

	Context("When Network is not created", func() {
		It("Should check that valid CR will be accepted", func() {
			testNamespaceName := createTestNamespace()

			crs := []v1alpha1.Subnet{
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "without-regions",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha1.SubnetSpec{
						CIDR: v1alpha1.CidrMustParse("127.0.0.0/24"),
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "with-cidr-rule",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha1.SubnetSpec{
						CIDR: v1alpha1.CidrMustParse("127.0.0.0/24"),
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []v1alpha1.Region{
							{
								Name:              "euw",
								AvailabilityZones: []string{"a"},
							},
							{
								Name:              "na",
								AvailabilityZones: []string{"a"},
							},
						},
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "with-capacity-rule",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha1.SubnetSpec{
						Capacity: resource.NewScaledQuantity(60, 0),
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []v1alpha1.Region{
							{
								Name:              "euw",
								AvailabilityZones: []string{"a", "b", "c"},
							},
						},
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "with-host-bits-rule",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha1.SubnetSpec{
						PrefixBits: bytePtr(20),
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []v1alpha1.Region{
							{
								Name:              "euw",
								AvailabilityZones: []string{"a"},
							},
						},
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "with-cidr-rule-and-without-parent-subnet",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha1.SubnetSpec{
						CIDR: v1alpha1.CidrMustParse("127.0.0.0/24"),
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []v1alpha1.Region{
							{
								Name:              "euw",
								AvailabilityZones: []string{"a"},
							},
						},
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "with-valid-consumer-ref",
						Namespace: testNamespaceName,
					},
					Spec: v1alpha1.SubnetSpec{
						CIDR: v1alpha1.CidrMustParse("127.0.0.0/24"),
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []v1alpha1.Region{
							{
								Name:              "euw",
								AvailabilityZones: []string{"a"},
							},
						},
						Consumer: &v1alpha1.ResourceReference{
							APIVersion: "sample.api/v1alpha1",
							Kind:       "SampleKind",
							Name:       "sample-name",
						},
					},
				},
			}

			ctx := context.Background()

			for _, cr := range crs {
				By(fmt.Sprintf("Attempting to create Subnet with valid configuration %s", cr.Name))
				Expect(k8sClient.Create(ctx, &cr)).Should(Succeed())
			}
		})
	})

	Context("When Subnet is created", func() {
		It("Should not allow to update CR", func() {
			testNamespaceName := createTestNamespace()

			By("Create Subnet CR")
			ctx := context.Background()

			testCidr, err := v1alpha1.CIDRFromString("10.0.0.0/8")
			Expect(err).NotTo(HaveOccurred())
			Expect(testCidr).NotTo(BeNil())

			cr := v1alpha1.Subnet{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-subnet",
					Namespace: testNamespaceName,
				},
				Spec: v1alpha1.SubnetSpec{
					CIDR: testCidr,
					ParentSubnet: corev1.LocalObjectReference{
						Name: "ps",
					},
					Network: corev1.LocalObjectReference{
						Name: "ng",
					},
					Regions: []v1alpha1.Region{
						{
							Name:              "euw",
							AvailabilityZones: []string{"a"},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, &cr)).Should(Succeed())
			Eventually(func() bool {
				namespacedName := types.NamespacedName{
					Namespace: cr.Namespace,
					Name:      cr.Name,
				}
				err := k8sClient.Get(ctx, namespacedName, &cr)
				return err == nil
			}, Timeout, Interval).Should(BeTrue())

			By("Try to update Subnet CR")
			cr.Spec.ParentSubnet.Name = "new"
			Expect(k8sClient.Update(ctx, &cr)).ShouldNot(Succeed())
		})
	})

	Context("When Subnet has sibling Subnets", func() {
		It("Can't be deleted", func() {
			testNamespaceName := createTestNamespace()
			By("Parent Subnet is created")
			ctx := context.Background()

			parentSubnetCidr, err := v1alpha1.CIDRFromString("10.0.0.0/8")
			Expect(err).NotTo(HaveOccurred())
			Expect(parentSubnetCidr).NotTo(BeNil())

			parentSubnet := v1alpha1.Subnet{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-parent-subnet",
					Namespace: testNamespaceName,
				},
				Spec: v1alpha1.SubnetSpec{
					CIDR: parentSubnetCidr,
					Network: corev1.LocalObjectReference{
						Name: "ng",
					},
					Regions: []v1alpha1.Region{
						{
							Name:              "euw",
							AvailabilityZones: []string{"a"},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, &parentSubnet)).Should(Succeed())
			Eventually(func() bool {
				namespacedName := types.NamespacedName{
					Namespace: parentSubnet.Namespace,
					Name:      parentSubnet.Name,
				}
				err := k8sClient.Get(ctx, namespacedName, &parentSubnet)
				return err == nil
			}, Timeout, Interval).Should(BeTrue())

			By("Child Subnet is created")
			childSubnetCidr, err := v1alpha1.CIDRFromString("10.0.0.0/16")
			Expect(err).NotTo(HaveOccurred())
			Expect(parentSubnetCidr).NotTo(BeNil())

			childSubnet := v1alpha1.Subnet{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-child-subnet",
					Namespace: testNamespaceName,
				},
				Spec: v1alpha1.SubnetSpec{
					CIDR: childSubnetCidr,
					ParentSubnet: corev1.LocalObjectReference{
						Name: parentSubnet.Name,
					},
					Network: corev1.LocalObjectReference{
						Name: "ng",
					},
					Regions: []v1alpha1.Region{
						{
							Name:              "euw",
							AvailabilityZones: []string{"a"},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, &childSubnet)).Should(Succeed())
			Eventually(func() bool {
				namespacedName := types.NamespacedName{
					Namespace: childSubnet.Namespace,
					Name:      childSubnet.Name,
				}
				err := k8sClient.Get(ctx, namespacedName, &childSubnet)
				return err == nil
			}, Timeout, Interval).Should(BeTrue())

			childSubnet.Status.State = v1alpha1.FinishedSubnetState
			Expect(k8sClient.Status().Update(ctx, &childSubnet)).Should(Succeed())
			Eventually(func() bool {
				childSubnetsMatchingFields := client.MatchingFields{
					FinishedChildSubnetToSubnetIndexKey: parentSubnet.Name,
				}
				subnets := &v1alpha1.SubnetList{}
				err := k8sClient.List(context.Background(), subnets, client.InNamespace(testNamespaceName), childSubnetsMatchingFields, client.Limit(1))
				if err != nil {
					return false
				}
				if len(subnets.Items) < 1 {
					return false
				}
				return true
			}, Timeout, Interval).Should(BeTrue())

			By("Deletion of parent Subnet is failed")
			Expect(k8sClient.Delete(ctx, &parentSubnet)).Should(Not(Succeed()))

			By("Child Subnet is deleted")
			Expect(k8sClient.Delete(ctx, &childSubnet)).Should(Succeed())
			Eventually(func() bool {
				namespacedName := types.NamespacedName{
					Namespace: childSubnet.Namespace,
					Name:      childSubnet.Name,
				}
				err := k8sClient.Get(ctx, namespacedName, &childSubnet)
				return apierrors.IsNotFound(err)
			}, Timeout, Interval).Should(BeTrue())

			By("Parent Subnet is deleted")
			Expect(k8sClient.Delete(ctx, &parentSubnet)).Should(Succeed())
		})
	})

	Context("When Subnet has sibling IPs", func() {
		It("Can't be deleted", func() {
			testNamespaceName := createTestNamespace()
			By("Parent Subnet is created")
			ctx := context.Background()

			parentSubnetCidr, err := v1alpha1.CIDRFromString("10.0.0.0/8")
			Expect(err).NotTo(HaveOccurred())
			Expect(parentSubnetCidr).NotTo(BeNil())

			parentSubnet := v1alpha1.Subnet{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-parent-subnet",
					Namespace: testNamespaceName,
				},
				Spec: v1alpha1.SubnetSpec{
					CIDR: parentSubnetCidr,
					ParentSubnet: corev1.LocalObjectReference{
						Name: "ps",
					},
					Network: corev1.LocalObjectReference{
						Name: "ng",
					},
					Regions: []v1alpha1.Region{
						{
							Name:              "euw",
							AvailabilityZones: []string{"a"},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, &parentSubnet)).Should(Succeed())
			Eventually(func() bool {
				namespacedName := types.NamespacedName{
					Namespace: parentSubnet.Namespace,
					Name:      parentSubnet.Name,
				}
				err := k8sClient.Get(ctx, namespacedName, &parentSubnet)
				return err == nil
			}, Timeout, Interval).Should(BeTrue())

			By("Child IP is created")
			childIPAddr, err := v1alpha1.IPAddrFromString("10.0.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(parentSubnetCidr).NotTo(BeNil())

			childIP := v1alpha1.IP{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-child-ip",
					Namespace: testNamespaceName,
				},
				Spec: v1alpha1.IPSpec{
					Subnet: corev1.LocalObjectReference{
						Name: parentSubnet.Name,
					},
					IP: childIPAddr,
				},
			}

			Expect(k8sClient.Create(ctx, &childIP)).Should(Succeed())
			Eventually(func() bool {
				namespacedName := types.NamespacedName{
					Namespace: childIP.Namespace,
					Name:      childIP.Name,
				}
				err := k8sClient.Get(ctx, namespacedName, &childIP)
				return err == nil
			}, Timeout, Interval).Should(BeTrue())

			childIP.Status.State = v1alpha1.FinishedIPState
			Expect(k8sClient.Status().Update(ctx, &childIP)).Should(Succeed())
			Eventually(func() bool {
				childIPsMatchingFields := client.MatchingFields{
					FinishedChildIPToSubnetIndexKey: parentSubnet.Name,
				}
				ips := &v1alpha1.IPList{}
				err := k8sClient.List(context.Background(), ips, client.InNamespace(testNamespaceName), childIPsMatchingFields, client.Limit(1))
				if err != nil {
					return false
				}
				if len(ips.Items) < 1 {
					return false
				}
				return true
			}, Timeout, Interval).Should(BeTrue())

			By("Deletion of parent Subnet is failed")
			Expect(k8sClient.Delete(ctx, &parentSubnet)).Should(Not(Succeed()))

			By("Child IP is deleted")
			Expect(k8sClient.Delete(ctx, &childIP)).Should(Succeed())
			Eventually(func() bool {
				namespacedName := types.NamespacedName{
					Namespace: childIP.Namespace,
					Name:      childIP.Name,
				}
				err := k8sClient.Get(ctx, namespacedName, &childIP)
				return apierrors.IsNotFound(err)
			}, Timeout, Interval).Should(BeTrue())

			By("Parent Subnet is deleted")
			Expect(k8sClient.Delete(ctx, &parentSubnet)).Should(Succeed())
		})
	})
})
