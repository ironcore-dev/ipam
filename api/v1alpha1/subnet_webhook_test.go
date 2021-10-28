package v1alpha1

import (
	"context"
	"fmt"
	"math"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

var _ = Describe("Subnet webhook", func() {
	const (
		SubnetNamespace = "default"
	)

	cidrMustParse := func(s string) *CIDR {
		cidr, err := CIDRFromString(s)
		Expect(err).NotTo(HaveOccurred())
		return cidr
	}

	bytePtr := func(b byte) *byte {
		return &b
	}

	Context("When Network is not created", func() {
		It("Should check that invalid CR will be rejected", func() {
			crs := []Subnet{
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "without-rules",
						Namespace: SubnetNamespace,
					},
					Spec: SubnetSpec{
						ParentSubnetName: "parent-subnet",
						NetworkName:      "parent-net",
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
						Namespace: SubnetNamespace,
					},
					Spec: SubnetSpec{
						CIDR:             cidrMustParse("127.0.0.0/24"),
						Capacity:         resource.NewScaledQuantity(60, 0),
						ParentSubnetName: "parent-subnet",
						NetworkName:      "parent-net",
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
						Namespace: SubnetNamespace,
					},
					Spec: SubnetSpec{
						Capacity:    resource.NewScaledQuantity(60, 0),
						NetworkName: "parent-net",
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
						Namespace: SubnetNamespace,
					},
					Spec: SubnetSpec{
						Capacity:    resource.NewScaledQuantity(0, 0),
						NetworkName: "parent-net",
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
						Namespace: SubnetNamespace,
					},
					Spec: SubnetSpec{
						Capacity:    resource.NewScaledQuantity(math.MaxInt64, resource.Exa),
						NetworkName: "parent-net",
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
						Namespace: SubnetNamespace,
					},
					Spec: SubnetSpec{
						CIDR:             cidrMustParse("127.0.0.0/24"),
						ParentSubnetName: "parent-subnet",
						NetworkName:      "parent-net",
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
						Namespace: SubnetNamespace,
					},
					Spec: SubnetSpec{
						CIDR:             cidrMustParse("127.0.0.0/24"),
						ParentSubnetName: "parent-subnet",
						NetworkName:      "parent-net",
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
			crs := []Subnet{
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "with-cidr-rule",
						Namespace: SubnetNamespace,
					},
					Spec: SubnetSpec{
						CIDR:             cidrMustParse("127.0.0.0/24"),
						ParentSubnetName: "parent-subnet",
						NetworkName:      "parent-net",
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
						Namespace: SubnetNamespace,
					},
					Spec: SubnetSpec{
						Capacity:         resource.NewScaledQuantity(60, 0),
						ParentSubnetName: "parent-subnet",
						NetworkName:      "parent-net",
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
						Namespace: SubnetNamespace,
					},
					Spec: SubnetSpec{
						PrefixBits:       bytePtr(20),
						ParentSubnetName: "parent-subnet",
						NetworkName:      "parent-net",
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
						Namespace: SubnetNamespace,
					},
					Spec: SubnetSpec{
						CIDR:        cidrMustParse("127.0.0.0/24"),
						NetworkName: "parent-net",
						Regions: []Region{
							{
								Name:              "euw",
								AvailabilityZones: []string{"a"},
							},
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
			By("Create Subnet CR")
			ctx := context.Background()

			testCidr, err := CIDRFromString("10.0.0.0/8")
			Expect(err).NotTo(HaveOccurred())
			Expect(testCidr).NotTo(BeNil())

			cr := Subnet{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "test-subnet",
					Namespace: SubnetNamespace,
				},
				Spec: SubnetSpec{
					CIDR:             testCidr,
					ParentSubnetName: "ps",
					NetworkName:      "ng",
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
			}).Should(BeTrue())

			By("Try to update Subnet CR")
			cr.Spec.ParentSubnetName = "new"
			Expect(k8sClient.Update(ctx, &cr)).ShouldNot(Succeed())
		})
	})
})
