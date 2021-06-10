package v1alpha1

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

var _ = Describe("Subnet webhook", func() {
	const (
		SubnetNamespace = "default"
	)

	Context("When Subnet is created", func() {
		It("Should not allow to update CR", func() {
			By(fmt.Sprintf("Create Subnet CR"))
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
					CIDR:              testCidr,
					ParentSubnetName:  "ps",
					NetworkName:       "ng",
					Regions:           []string{"euw"},
					AvailabilityZones: []string{"a"},
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

			By(fmt.Sprintf("Try to update Subnet CR"))
			cr.Spec.ParentSubnetName = "new"
			Expect(k8sClient.Update(ctx, &cr)).ShouldNot(Succeed())
		})
	})
})
