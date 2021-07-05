package v1alpha1

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("IP webhook", func() {
	cidrMustParse := func(cidrString string) *CIDR {
		cidr, err := CIDRFromString(cidrString)
		if err != nil {
			panic(err)
		}
		return cidr
	}
	ipMustParse := func(ipString string) *IP {
		ip, err := IPFromString(ipString)
		if err != nil {
			panic(err)
		}
		return ip
	}

	Context("IP webhook test", func() {
		It("Should fail with nonexistent related CRD", func() {
			ctx := context.Background()
			ip := &Ip{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ApiVersion,
					Kind:       "Ip",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ip0",
					Namespace: Namespace,
				},
				Spec: IpSpec{
					Subnet: "subnet1",
					CRD: &CRD{
						GroupVersion: "v1",
						Kind:         "ConfigMap",
						Name:         "configmap-that-doesnt-exist",
					},
					IP: ipMustParse("1.12.12.123"),
				},
			}
			Expect(k8sClient.Create(ctx, ip)).ShouldNot(Succeed())
		})

		It("Should allocate free IP", func() {
			referredResource := &NetworkCounter{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "referred-networkcounter",
					Namespace: Namespace,
				},
				Spec: NetworkCounterSpec{},
			}
			By("Expecting referred resource to be created successfully")
			Expect(k8sClient.Create(ctx, referredResource)).Should(Succeed())

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
					CIDR:              cidrMustParse("10.12.34.0/24"),
					NetworkName:       "ng1",
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
					CIDR:              cidrMustParse("10.12.34.0/26"),
					ParentSubnetName:  "subnet1",
					NetworkName:       "ng1",
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
					CIDR:              cidrMustParse("10.12.34.128/25"),
					ParentSubnetName:  "subnet1",
					NetworkName:       "ng1",
					Regions:           []string{"euw"},
					AvailabilityZones: []string{"a"},
				},
				Status: SubnetStatus{
					Type: CIPv4SubnetType,
				},
			}
			By("Expecting Subnet 3 Create Successful")
			Expect(k8sClient.Create(ctx, subnet3)).Should(Succeed())

			ip := &Ip{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ApiVersion,
					Kind:       "Ip",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ip1",
					Namespace: Namespace,
				},
				Spec: IpSpec{
					Subnet: "subnet1",
					CRD: &CRD{
						GroupVersion: ApiVersion,
						Kind:         "NetworkCounter",
						Name:         "referred-networkcounter",
					},
				},
			}
			By("Expecting Ip Create Successful")
			Expect(k8sClient.Create(ctx, ip)).Should(Succeed())

			key := types.NamespacedName{
				Name:      "ip1",
				Namespace: Namespace,
			}
			Eventually(func() bool {
				ip := &Ip{}
				_ = k8sClient.Get(context.Background(), key, ip)
				return ip.Spec.IP == ipMustParse("10.12.34.64")
			}, timeout, interval).Should(BeTrue())
		})

		It("Should create without CRD specified", func() {
			ctx := context.Background()
			ip := &Ip{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ApiVersion,
					Kind:       "Ip",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ip2",
					Namespace: Namespace,
				},
				Spec: IpSpec{
					Subnet: "subnet1",
					IP:     ipMustParse("0.0.0.1"),
				},
			}
			By("Expecting Ip Create Successful")
			Expect(k8sClient.Create(ctx, ip)).Should(Succeed())
		})

		It("Should not allow to use already allocated IP", func() {
			ctx := context.Background()
			ip := &Ip{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ApiVersion,
					Kind:       "Ip",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ip2",
					Namespace: Namespace,
				},
				Spec: IpSpec{
					Subnet: "subnet1",
					CRD: &CRD{
						GroupVersion: ApiVersion,
						Kind:         "NetworkCounter",
						Name:         "referred-networkcounter",
					},
					IP: ipMustParse("10.12.34.64"),
				},
			}
			By("Expecting Ip Create Successful")
			Expect(k8sClient.Create(ctx, ip)).ShouldNot(Succeed())
		})

		It("Should not allow to use IP from child subnet", func() {
			ctx := context.Background()
			ip := &Ip{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ApiVersion,
					Kind:       "Ip",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ip3",
					Namespace: Namespace,
				},
				Spec: IpSpec{
					Subnet: "subnet1",
					CRD: &CRD{
						GroupVersion: ApiVersion,
						Kind:         "NetworkCounter",
						Name:         "referred-networkcounter",
					},
					IP: ipMustParse("10.12.34.255"),
				},
			}
			By("Expecting Ip Create Successful")
			Expect(k8sClient.Create(ctx, ip)).ShouldNot(Succeed())
		})
	})
})
