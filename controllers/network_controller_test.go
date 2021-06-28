package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/onmetal/ipam/api/v1alpha1"
)

var _ = Describe("Network controller", func() {
	const (
		VXLANNetworkName  = "test-vxlan-network"
		GENEVENetworkName = "test-geneve-network"
		MPLSNetworkName   = "test-mpls-network"

		CopyPostfix = "-copy"

		NetworkNamespace = "default"

		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)

	AfterEach(func() {
		ctx := context.Background()
		resources := []struct {
			res   client.Object
			list  client.ObjectList
			count func(client.ObjectList) int
		}{
			{
				res:  &v1alpha1.Network{},
				list: &v1alpha1.NetworkList{},
				count: func(objList client.ObjectList) int {
					list := objList.(*v1alpha1.NetworkList)
					return len(list.Items)
				},
			},
		}

		for _, r := range resources {
			Expect(k8sClient.DeleteAllOf(ctx, r.res, client.InNamespace(NetworkNamespace))).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.List(ctx, r.list)
				if err != nil {
					return false
				}
				if r.count(r.list) > 0 {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
		}
	})

	Context("When network CR is created", func() {
		It("Should get ID assigned if ID is vacant", func() {
			testNetworkCases := []struct {
				counterName string
				firstId     *v1alpha1.NetworkID
				network     *v1alpha1.Network
			}{
				{
					counterName: CVXLANCounterName,
					firstId:     v1alpha1.CVXLANFirstAvaliableID,
					network: &v1alpha1.Network{
						ObjectMeta: metav1.ObjectMeta{
							Name:      VXLANNetworkName,
							Namespace: NetworkNamespace,
						},
						Spec: v1alpha1.NetworkSpec{
							Type: v1alpha1.CVXLANNetworkType,
						},
					},
				},
				{
					counterName: CGENEVECounterName,
					firstId:     v1alpha1.CGENEVEFirstAvaliableID,
					network: &v1alpha1.Network{
						ObjectMeta: metav1.ObjectMeta{
							Name:      GENEVENetworkName,
							Namespace: NetworkNamespace,
						},
						Spec: v1alpha1.NetworkSpec{
							Type: v1alpha1.CGENEVENetworkType,
						},
					},
				},
				{
					counterName: CMPLSCounterName,
					firstId:     v1alpha1.CMPLSFirstAvailableID,
					network: &v1alpha1.Network{
						ObjectMeta: metav1.ObjectMeta{
							Name:      MPLSNetworkName,
							Namespace: NetworkNamespace,
						},
						Spec: v1alpha1.NetworkSpec{
							Type: v1alpha1.CMPLSNetworkType,
						},
					},
				},
			}

			ctx := context.Background()

			for _, testNetworkCase := range testNetworkCases {
				testNetwork := testNetworkCase.network
				networkNamespacedName := types.NamespacedName{
					Namespace: testNetwork.Namespace,
					Name:      testNetwork.Name,
				}

				By(fmt.Sprintf("%s network CR is installed first time", testNetworkCase.network.Spec.Type))
				Expect(k8sClient.Create(ctx, testNetwork)).To(Succeed())

				Eventually(func() bool {
					err := k8sClient.Get(ctx, networkNamespacedName, testNetwork)
					if err != nil {
						return false
					}
					if !controllerutil.ContainsFinalizer(testNetwork, CNetworkFinalizer) {
						return false
					}
					if testNetwork.Status.State != v1alpha1.CFinishedRequestState {
						return false
					}
					if testNetwork.Status.Reserved == nil {
						return false
					}
					return true
				}, timeout, interval).Should(BeTrue())

				By(fmt.Sprintf("%s network counter is created", testNetworkCase.network.Spec.Type))
				counter := v1alpha1.NetworkCounter{}
				networkCounterNamespacedName := types.NamespacedName{
					Namespace: testNetwork.Namespace,
					Name:      testNetworkCase.counterName,
				}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, networkCounterNamespacedName, &counter)
					if err != nil {
						return false
					}
					return true
				}, timeout, interval).Should(BeTrue())

				By(fmt.Sprintf("%s network ID reserved in counter", testNetworkCase.network.Spec.Type))
				Expect(v1alpha1.NewNetworkCounterSpec(testNetwork.Spec.Type).CanReserve(testNetwork.Status.Reserved)).Should(BeTrue())
				Expect(counter.Spec.CanReserve(testNetwork.Status.Reserved)).Should(BeFalse())

				By(fmt.Sprintf("%s network ID with the same ID is created", testNetworkCase.network.Spec.Type))
				testNetworkCopy := v1alpha1.Network{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testNetwork.Name + CopyPostfix,
						Namespace: testNetwork.Namespace,
					},
					Spec: *testNetwork.Spec.DeepCopy(),
				}
				testNetworkCopy.Spec.ID = testNetwork.Status.Reserved
				Expect(k8sClient.Create(ctx, &testNetworkCopy)).Should(Succeed())

				By(fmt.Sprintf("%s network ID with the same ID fails on ID reservation", testNetworkCase.network.Spec.Type))
				Eventually(func() bool {
					networkCopyNamespacedName := types.NamespacedName{
						Namespace: testNetworkCopy.Namespace,
						Name:      testNetworkCopy.Name,
					}
					err := k8sClient.Get(ctx, networkCopyNamespacedName, &testNetworkCopy)
					if err != nil {
						return false
					}
					if testNetworkCopy.Status.State != v1alpha1.CFailedRequestState {
						return false
					}
					return true
				}, timeout, interval).Should(BeTrue())

				By(fmt.Sprintf("%s network ID CR deleted", testNetworkCase.network.Spec.Type))
				oldNetworkID := testNetwork.Status.Reserved.DeepCopy()
				Expect(k8sClient.Delete(ctx, testNetwork)).Should(Succeed())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, networkNamespacedName, testNetwork)
					if apierrors.IsNotFound(err) {
						return true
					}
					return false
				}, timeout, interval).Should(BeTrue())

				By(fmt.Sprintf("%s network ID released", testNetworkCase.network.Spec.Type))
				Eventually(func() bool {
					err := k8sClient.Get(ctx, networkCounterNamespacedName, &counter)
					if err != nil {
						return false
					}
					return true
				}, timeout, interval).Should(BeTrue())

				Expect(counter.Spec.CanReserve(oldNetworkID)).Should(BeTrue())
			}
		})
	})
})
