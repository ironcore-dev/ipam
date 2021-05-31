package v1alpha1

import (
	"context"
	subnetv1alpha1 "github.com/onmetal/k8s-subnet/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("IPAM webhook", func() {
	Context("IPAM webhook test", func() {
		It("Should allocate free IP", func() {

			example := &Example{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ApiVersion,
					Kind:       "Example",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example1",
					Namespace: Namespace,
				},
				Spec: ExampleSpec{
					Foo: "bar",
				},
			}
			By("Expecting Example Create Successful")
			Expect(k8sClient.Create(ctx, example)).Should(Succeed())

			subnet := &subnetv1alpha1.Subnet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ApiVersion,
					Kind:       "Subnet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "subnet1",
					Namespace: Namespace,
				},
				Spec: subnetv1alpha1.SubnetSpec{
					Type: "ipv4",
					CIDR: "10.12.34.0/24",
				},
			}
			By("Expecting Subnet 1 Create Successful")
			Expect(k8sClient.Create(ctx, subnet)).Should(Succeed())

			subnet2 := &subnetv1alpha1.Subnet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ApiVersion,
					Kind:       "Subnet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "subnet2",
					Namespace: Namespace,
				},
				Spec: subnetv1alpha1.SubnetSpec{
					Type:           "ipv4",
					CIDR:           "10.12.34.0/26",
					SubnetParentID: "subnet1",
				},
			}
			By("Expecting Subnet 2 Create Successful")
			Expect(k8sClient.Create(ctx, subnet2)).Should(Succeed())

			subnet3 := &subnetv1alpha1.Subnet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ApiVersion,
					Kind:       "Subnet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "subnet3",
					Namespace: Namespace,
				},
				Spec: subnetv1alpha1.SubnetSpec{
					Type:           "ipv4",
					CIDR:           "10.12.34.128/25",
					SubnetParentID: "subnet1",
				},
			}
			By("Expecting Subnet 3 Create Successful")
			Expect(k8sClient.Create(ctx, subnet3)).Should(Succeed())

			ipam := &Ipam{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ApiVersion,
					Kind:       "Ipam",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ipam1",
					Namespace: Namespace,
				},
				Spec: IpamSpec{
					Subnet: "subnet1",
					CRD:    "example1",
				},
			}
			By("Expecting Ipam Create Successful")
			Expect(k8sClient.Create(ctx, ipam)).Should(Succeed())

			key := types.NamespacedName{
				Name:      "ipam1",
				Namespace: Namespace,
			}
			Eventually(func() bool {
				ipam := &Ipam{}
				_ = k8sClient.Get(context.Background(), key, ipam)
				return ipam.Spec.IP == "10.12.34.64"
			}, timeout, interval).Should(BeTrue())
		})

		It("Should not allow to use already allocated IP", func() {
			ctx := context.Background()
			ipam := &Ipam{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ApiVersion,
					Kind:       "Ipam",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ipam2",
					Namespace: Namespace,
				},
				Spec: IpamSpec{
					Subnet: "subnet1",
					CRD:    "example1",
					IP:     "10.12.34.64",
				},
			}
			By("Expecting Ipam Create Successful")
			Expect(k8sClient.Create(ctx, ipam)).ShouldNot(Succeed())
		})

		It("Should not allow to use IP from child subnet", func() {
			ctx := context.Background()
			ipam := &Ipam{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ApiVersion,
					Kind:       "Ipam",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ipam3",
					Namespace: Namespace,
				},
				Spec: IpamSpec{
					Subnet: "subnet1",
					CRD:    "example1",
					IP:     "10.12.34.255",
				},
			}
			By("Expecting Ipam Create Successful")
			Expect(k8sClient.Create(ctx, ipam)).ShouldNot(Succeed())
		})
	})
})
