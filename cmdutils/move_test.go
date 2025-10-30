// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package cmdutils

import (
	"context"
	"log/slog"

	ipamv1alphav1 "github.com/ironcore-dev/ipam/api/ipam/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

const (
	ns = "test-namespace"
)

func namedObj[T client.Object](obj T, name string) T {
	obj.SetName(name)
	obj.SetNamespace(ns)
	return obj
}

func create[T client.Object](ctx SpecContext, cl client.Client, obj T) T {
	Expect(cl.Create(ctx, obj)).To(Succeed())
	Eventually(func(g Gomega) error {
		return clients.Source.Get(ctx, client.ObjectKeyFromObject(obj), obj)
	}).Should(Succeed())
	return obj
}

var _ = Describe("ipamctl move", func() {
	It("Should successfully move IPAM CRs with from a source cluster on a target cluster", func(ctx SpecContext) {
		slog.SetLogLoggerLevel(slog.LevelDebug)

		// source cluster setup
		create(ctx, clients.Source, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})

		testIP := "192.168.0.0"
		testCidr, _ := ipamv1alphav1.CIDRFromString(testIP + "/24")

		sourceNetwork := create(ctx, clients.Source, namedObj(&ipamv1alphav1.Network{}, "network"))

		sourceSubnet := namedObj(&ipamv1alphav1.Subnet{}, "subnet")
		sourceSubnet.Spec.Network.Name = sourceNetwork.Name
		sourceSubnet.Spec.CIDR = testCidr
		sourceSubnet = create(ctx, clients.Source, sourceSubnet)

		sourceIP := namedObj(&ipamv1alphav1.IP{}, "ip")
		sourceIP.Spec.Subnet.Name = sourceSubnet.Name
		sourceIP.Spec.IP = ipamv1alphav1.IPMustParse(testIP)
		sourceIP = create(ctx, clients.Source, sourceIP)

		// target cluster setup
		create(ctx, clients.Target, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})
		targetNetwork := namedObj(&ipamv1alphav1.Network{}, sourceNetwork.Name)
		targetSubnet := namedObj(&ipamv1alphav1.Subnet{}, sourceSubnet.Name)
		targetIP := namedObj(&ipamv1alphav1.IP{}, sourceIP.Name)

		// TEST
		crsSchema := []schema.GroupVersionKind{}
		for _, crdKind := range []string{"Network", "Subnet", "IP"} {
			crsSchema = append(crsSchema,
				schema.GroupVersionKind{Group: "ipam.metal.ironcore.dev", Version: "v1alpha1", Kind: crdKind})
		}
		err := Move(context.TODO(), clients, crsSchema, "", false)
		Expect(err).ToNot(HaveOccurred())

		SetClient(clients.Target)

		Eventually(Get(targetNetwork)).Should(Succeed())

		Eventually(Get(targetSubnet)).Should(Succeed())
		Expect(targetSubnet.Spec.Network.Name).To(Equal(targetNetwork.Name))
		Expect(targetSubnet.Spec.CIDR).To(Equal(testCidr))

		Eventually(Get(targetIP)).Should(Succeed())
		Expect(targetIP.Spec.Subnet.Name).To(Equal(targetSubnet.Name))
		Expect(targetIP.Spec.IP).To(Equal(ipamv1alphav1.IPMustParse(testIP)))
	})
})
