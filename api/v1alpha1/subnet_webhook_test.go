// Copyright 2023 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"context"
	"fmt"
	"math"

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

			crs := []Subnet{
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "without-rules",
						Namespace: testNamespaceName,
					},
					Spec: SubnetSpec{
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []Region{
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
					Spec: SubnetSpec{
						CIDR:     cidrMustParse("127.0.0.0/24"),
						Capacity: resource.NewScaledQuantity(60, 0),
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []Region{
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
					Spec: SubnetSpec{
						Capacity: resource.NewScaledQuantity(60, 0),
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []Region{
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
					Spec: SubnetSpec{
						Capacity: resource.NewScaledQuantity(0, 0),
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []Region{
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
					Spec: SubnetSpec{
						Capacity: resource.NewScaledQuantity(math.MaxInt64, resource.Exa),
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []Region{
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
					Spec: SubnetSpec{
						CIDR: cidrMustParse("127.0.0.0/24"),
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []Region{
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
					Spec: SubnetSpec{
						CIDR: cidrMustParse("127.0.0.0/24"),
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []Region{
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
					Spec: SubnetSpec{
						CIDR: cidrMustParse("127.0.0.0/24"),
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []Region{
							{
								Name:              "euw",
								AvailabilityZones: []string{"a"},
							},
						},
						Consumer: &ResourceReference{
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

			crs := []Subnet{
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "without-regions",
						Namespace: testNamespaceName,
					},
					Spec: SubnetSpec{
						CIDR: cidrMustParse("127.0.0.0/24"),
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
					Spec: SubnetSpec{
						CIDR: cidrMustParse("127.0.0.0/24"),
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []Region{
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
					Spec: SubnetSpec{
						Capacity: resource.NewScaledQuantity(60, 0),
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []Region{
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
					Spec: SubnetSpec{
						PrefixBits: bytePtr(20),
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []Region{
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
					Spec: SubnetSpec{
						CIDR: cidrMustParse("127.0.0.0/24"),
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []Region{
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
					Spec: SubnetSpec{
						CIDR: cidrMustParse("127.0.0.0/24"),
						ParentSubnet: corev1.LocalObjectReference{
							Name: "parent-subnet",
						},
						Network: corev1.LocalObjectReference{
							Name: "parent-net",
						},
						Regions: []Region{
							{
								Name:              "euw",
								AvailabilityZones: []string{"a"},
							},
						},
						Consumer: &ResourceReference{
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

			testCidr, err := CIDRFromString("10.0.0.0/8")
			Expect(err).NotTo(HaveOccurred())
			Expect(testCidr).NotTo(BeNil())

			cr := Subnet{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-subnet",
					Namespace: testNamespaceName,
				},
				Spec: SubnetSpec{
					CIDR: testCidr,
					ParentSubnet: corev1.LocalObjectReference{
						Name: "ps",
					},
					Network: corev1.LocalObjectReference{
						Name: "ng",
					},
					Regions: []Region{
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
				if err != nil {
					return false
				}
				return true
			}, CTimeout, CInterval).Should(BeTrue())

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

			parentSubnetCidr, err := CIDRFromString("10.0.0.0/8")
			Expect(err).NotTo(HaveOccurred())
			Expect(parentSubnetCidr).NotTo(BeNil())

			parentSubnet := Subnet{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-parent-subnet",
					Namespace: testNamespaceName,
				},
				Spec: SubnetSpec{
					CIDR: parentSubnetCidr,
					Network: corev1.LocalObjectReference{
						Name: "ng",
					},
					Regions: []Region{
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
				if err != nil {
					return false
				}
				return true
			}, CTimeout, CInterval).Should(BeTrue())

			By("Child Subnet is created")
			childSubnetCidr, err := CIDRFromString("10.0.0.0/16")
			Expect(err).NotTo(HaveOccurred())
			Expect(parentSubnetCidr).NotTo(BeNil())

			childSubnet := Subnet{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-child-subnet",
					Namespace: testNamespaceName,
				},
				Spec: SubnetSpec{
					CIDR: childSubnetCidr,
					ParentSubnet: corev1.LocalObjectReference{
						Name: parentSubnet.Name,
					},
					Network: corev1.LocalObjectReference{
						Name: "ng",
					},
					Regions: []Region{
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
				if err != nil {
					return false
				}
				return true
			}, CTimeout, CInterval).Should(BeTrue())

			childSubnet.Status.State = CFinishedSubnetState
			Expect(k8sClient.Status().Update(ctx, &childSubnet)).Should(Succeed())
			Eventually(func() bool {
				childSubnetsMatchingFields := client.MatchingFields{
					CFinishedChildSubnetToSubnetIndexKey: parentSubnet.Name,
				}
				subnets := &SubnetList{}
				err := subnetWebhookClient.List(context.Background(), subnets, client.InNamespace(testNamespaceName), childSubnetsMatchingFields, client.Limit(1))
				if err != nil {
					return false
				}
				if len(subnets.Items) < 1 {
					return false
				}
				return true
			}, CTimeout, CInterval).Should(BeTrue())

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
				if !apierrors.IsNotFound(err) {
					return false
				}
				return true
			}, CTimeout, CInterval).Should(BeTrue())

			By("Parent Subnet is deleted")
			Expect(k8sClient.Delete(ctx, &parentSubnet)).Should(Succeed())
		})
	})

	Context("When Subnet has sibling IPs", func() {
		It("Can't be deleted", func() {
			testNamespaceName := createTestNamespace()
			By("Parent Subnet is created")
			ctx := context.Background()

			parentSubnetCidr, err := CIDRFromString("10.0.0.0/8")
			Expect(err).NotTo(HaveOccurred())
			Expect(parentSubnetCidr).NotTo(BeNil())

			parentSubnet := Subnet{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-parent-subnet",
					Namespace: testNamespaceName,
				},
				Spec: SubnetSpec{
					CIDR: parentSubnetCidr,
					ParentSubnet: corev1.LocalObjectReference{
						Name: "ps",
					},
					Network: corev1.LocalObjectReference{
						Name: "ng",
					},
					Regions: []Region{
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
				if err != nil {
					return false
				}
				return true
			}, CTimeout, CInterval).Should(BeTrue())

			By("Child IP is created")
			childIPAddr, err := IPAddrFromString("10.0.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(parentSubnetCidr).NotTo(BeNil())

			childIP := IP{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-child-ip",
					Namespace: testNamespaceName,
				},
				Spec: IPSpec{
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
				if err != nil {
					return false
				}
				return true
			}, CTimeout, CInterval).Should(BeTrue())

			childIP.Status.State = CFinishedIPState
			Expect(k8sClient.Status().Update(ctx, &childIP)).Should(Succeed())
			Eventually(func() bool {
				childIPsMatchingFields := client.MatchingFields{
					CFinishedChildIPToSubnetIndexKey: parentSubnet.Name,
				}
				ips := &IPList{}
				err := subnetWebhookClient.List(context.Background(), ips, client.InNamespace(testNamespaceName), childIPsMatchingFields, client.Limit(1))
				if err != nil {
					return false
				}
				if len(ips.Items) < 1 {
					return false
				}
				return true
			}, CTimeout, CInterval).Should(BeTrue())

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
				if !apierrors.IsNotFound(err) {
					return false
				}
				return true
			}, CTimeout, CInterval).Should(BeTrue())

			By("Parent Subnet is deleted")
			Expect(k8sClient.Delete(ctx, &parentSubnet)).Should(Succeed())
		})
	})
})
