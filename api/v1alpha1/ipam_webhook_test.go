package v1alpha1

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("IPAM webhook", func() {
	cidrMustParse := func(cidrString string) *CIDR {
		cidr, err := CIDRFromString(cidrString)
		if err != nil {
			panic(err)
		}
		return cidr
	}

	Context("IPAM webhook test", func() {
		It("Should fail with nonexistent related CRD", func() {
			ctx := context.Background()
			ipam := &Ipam{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ApiVersion,
					Kind:       "Ipam",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ipam0",
					Namespace: Namespace,
				},
				Spec: IpamSpec{
					Subnet: "subnet1",
					CRD: &CRD{
						GroupVersion: ApiVersion,
						Kind:         "Example",
						Name:         "example2",
					},
					IP: "1.12.12.123",
				},
			}
			Expect(k8sClient.Create(ctx, ipam)).ShouldNot(Succeed())
		})

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

			subnet := &Subnet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ApiVersion,
					Kind:       "Subnet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "subnet1",
					Namespace: Namespace,
				},
				Spec: SubnetSpec{
					CIDR:              *cidrMustParse("10.12.34.0/24"),
					NetworkGlobalName: "ng1",
					Regions:           []string{"euw"},
					AvailabilityZones: []string{"a"},
				},
				Status: SubnetStatus{
					Type: CIPv4SubnetType,
				},
			}
			By("Expecting Subnet 1 Create Successful")
			Expect(k8sClient.Create(ctx, subnet)).Should(Succeed())

			subnet2 := &Subnet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ApiVersion,
					Kind:       "Subnet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "subnet2",
					Namespace: Namespace,
				},
				Spec: SubnetSpec{
					CIDR:              *cidrMustParse("10.12.34.0/26"),
					ParentSubnetName:  "subnet1",
					NetworkGlobalName: "ng1",
					Regions:           []string{"euw"},
					AvailabilityZones: []string{"a"},
				},
				Status: SubnetStatus{
					Type: CIPv4SubnetType,
				},
			}
			By("Expecting Subnet 2 Create Successful")
			Expect(k8sClient.Create(ctx, subnet2)).Should(Succeed())

			subnet3 := &Subnet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ApiVersion,
					Kind:       "Subnet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "subnet3",
					Namespace: Namespace,
				},
				Spec: SubnetSpec{
					CIDR:              *cidrMustParse("10.12.34.128/25"),
					ParentSubnetName:  "subnet1",
					NetworkGlobalName: "ng1",
					Regions:           []string{"euw"},
					AvailabilityZones: []string{"a"},
				},
				Status: SubnetStatus{
					Type: CIPv4SubnetType,
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
					CRD: &CRD{
						GroupVersion: ApiVersion,
						Kind:         "Example",
						Name:         "example1",
					},
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

		It("Should create without CRD specified", func() {
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
					IP:     "0.0.0.1",
				},
			}
			By("Expecting Ipam Create Successful")
			Expect(k8sClient.Create(ctx, ipam)).ShouldNot(Succeed())
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
					CRD: &CRD{
						GroupVersion: ApiVersion,
						Kind:         "Example",
						Name:         "example1",
					},
					IP: "10.12.34.64",
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
					CRD: &CRD{
						GroupVersion: ApiVersion,
						Kind:         "Example",
						Name:         "example1",
					},
					IP: "10.12.34.255",
				},
			}
			By("Expecting Ipam Create Successful")
			Expect(k8sClient.Create(ctx, ipam)).ShouldNot(Succeed())
		})
	})
})
