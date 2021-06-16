package v1alpha1

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

var _ = Describe("Network webhook", func() {
	const (
		NetworkNamespace = "default"
	)

	Context("When Network is not created", func() {
		It("Should check that invalid CR will be rejected", func() {
			crs := []Network{
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "empty-type",
						Namespace: NetworkNamespace,
					},
					Spec: NetworkSpec{
						ID: NetworkIDFromInt64(1000),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-vxlan-1",
						Namespace: NetworkNamespace,
					},
					Spec: NetworkSpec{
						ID:   NetworkIDFromBytes([]byte{0}),
						Type: CVXLANNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-vxlan-2",
						Namespace: NetworkNamespace,
					},
					Spec: NetworkSpec{
						ID:   NetworkIDFromBytes([]byte{99}),
						Type: CVXLANNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-vxlan-3",
						Namespace: NetworkNamespace,
					},
					Spec: NetworkSpec{
						ID:   NetworkIDFromBytes([]byte{0, 0, 0, 1}),
						Type: CVXLANNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-mpls-1",
						Namespace: NetworkNamespace,
					},
					Spec: NetworkSpec{
						ID:   NetworkIDFromBytes([]byte{0}),
						Type: CMPLSNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "out-of-range-mpls-1",
						Namespace: NetworkNamespace,
					},
					Spec: NetworkSpec{
						ID:   NetworkIDFromBytes([]byte{15}),
						Type: CMPLSNetworkType,
					},
				},
			}

			ctx := context.Background()

			for _, cr := range crs {
				By(fmt.Sprintf("Attempting to create Network with invalid configuration %s", cr.Name))
				Expect(k8sClient.Create(ctx, &cr)).ShouldNot(Succeed())
			}
		})
	})

	Context("When Network is not created", func() {
		It("Should check that valid CR will be accepted", func() {
			crs := []Network{
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "empty-type-and-id",
						Namespace: NetworkNamespace,
					},
					Spec: NetworkSpec{},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "vxlan-no-id",
						Namespace: NetworkNamespace,
					},
					Spec: NetworkSpec{
						Type: CVXLANNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "vxlan-border-bottom",
						Namespace: NetworkNamespace,
					},
					Spec: NetworkSpec{
						Type: CVXLANNetworkType,
						ID:   NetworkIDFromBytes([]byte{100}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "vxlan-border-top",
						Namespace: NetworkNamespace,
					},
					Spec: NetworkSpec{
						Type: CVXLANNetworkType,
						ID:   NetworkIDFromBytes([]byte{255, 255, 255}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "vxlan-middle-val",
						Namespace: NetworkNamespace,
					},
					Spec: NetworkSpec{
						Type: CVXLANNetworkType,
						ID:   NetworkIDFromBytes([]byte{255, 16}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "mpls-no-id",
						Namespace: NetworkNamespace,
					},
					Spec: NetworkSpec{
						Type: CMPLSNetworkType,
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "mpls-border-num",
						Namespace: NetworkNamespace,
					},
					Spec: NetworkSpec{
						Type: CMPLSNetworkType,
						ID:   NetworkIDFromBytes([]byte{16}),
					},
				},
				{
					ObjectMeta: controllerruntime.ObjectMeta{
						Name:      "mpls-big-num",
						Namespace: NetworkNamespace,
					},
					Spec: NetworkSpec{
						Type: CMPLSNetworkType,
						ID:   NetworkIDFromBytes([]byte{1, 11, 12, 13, 14, 15, 16}),
					},
				},
			}

			ctx := context.Background()

			for _, cr := range crs {
				By(fmt.Sprintf("Attempting to create Network with valid configuration %s", cr.Name))
				Expect(k8sClient.Create(ctx, &cr)).Should(Succeed())
			}
		})
	})

	Context("When Network is created with ID and Type", func() {
		It("Should not allow to update CR", func() {
			cr := Network{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "network-with-id-and-type-failed-to-update",
					Namespace: NetworkNamespace,
				},
				Spec: NetworkSpec{
					Type: CMPLSNetworkType,
					ID:   NetworkIDFromBytes([]byte{1, 11, 12, 13, 14, 15, 16}),
				},
			}

			By(fmt.Sprintf("Create network CR"))
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

			By(fmt.Sprintf("Try to update network CR"))
			cr.Spec.ID = NetworkIDFromInt64(10)
			Expect(k8sClient.Update(ctx, &cr)).ShouldNot(Succeed())
		})
	})

	Context("When Network is created with Type", func() {
		It("Should not allow to update CR", func() {
			cr := Network{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "network-with-type-failed-to-update",
					Namespace: NetworkNamespace,
				},
				Spec: NetworkSpec{
					Type: CMPLSNetworkType,
				},
			}

			By(fmt.Sprintf("Create network CR"))
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

			By(fmt.Sprintf("Try to update network CR"))
			cr.Spec.ID = NetworkIDFromInt64(10)
			Expect(k8sClient.Update(ctx, &cr)).ShouldNot(Succeed())
		})
	})

	Context("When Network is created without ID and Type", func() {
		It("Should allow to update CR with ID and Type", func() {
			By(fmt.Sprintf("Create network CR"))
			cr := Network{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "network-without-id-and-type-succeed-to-update",
					Namespace: NetworkNamespace,
				},
				Spec: NetworkSpec{},
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

			By(fmt.Sprintf("Update empty network CR ID and Type with values"))
			cr.Spec.ID = NetworkIDFromBytes([]byte{1, 11, 12, 13, 14, 15, 16})
			cr.Spec.Type = CMPLSNetworkType

			Expect(k8sClient.Update(ctx, &cr)).Should(Succeed())
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

			By(fmt.Sprintf("Update network CR description"))
			cr.Spec.Description = "sample description"
			Expect(k8sClient.Update(ctx, &cr)).Should(Succeed())
		})
	})
})
