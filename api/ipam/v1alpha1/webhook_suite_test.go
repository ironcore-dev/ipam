// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	// +kubebuilder:scaffold:imports

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

const (
	CTestNamespacePrefix = "test-ns-"
	CTimeout             = time.Second * 30
	CInterval            = time.Millisecond * 250
)

var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Webhook Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: false,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "..", "config", "webhook")},
		},
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = admissionv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = admissionv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	// start webhook server using Manager
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    webhookInstallOptions.LocalServingPort,
			Host:    webhookInstallOptions.LocalServingHost,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
		LeaderElection: false,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	Expect(err).NotTo(HaveOccurred())

	err = (&NetworkCounter{}).SetupWebhookWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	err = (&Network{}).SetupWebhookWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	err = (&Subnet{}).SetupWebhookWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	err = (&IP{}).SetupWebhookWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:webhook

	go func() {
		err = mgr.Start(ctx)
		if err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
	}()

	// wait for the webhook server to get ready
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}
		_ = conn.Close()
		return nil
	}).Should(Succeed())

	k8sClient = mgr.GetClient()
	Expect(k8sClient).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func createTestNamespace() string {
	testNamespace := &v1.Namespace{
		ObjectMeta: ctrl.ObjectMeta{
			GenerateName: CTestNamespacePrefix,
		},
	}
	Expect(k8sClient.Create(ctx, testNamespace)).To(Succeed())
	testNamespaceNamespacedName := types.NamespacedName{
		Name: testNamespace.Name,
	}
	Eventually(func() bool {
		if err := k8sClient.Get(ctx, testNamespaceNamespacedName, testNamespace); err != nil {
			return false
		}
		return true
	}, CTimeout, CInterval).Should(BeTrue())

	return testNamespace.Name
}

func ipMustParse(ipString string) *IPAddr {
	ip, err := IPAddrFromString(ipString)
	if err != nil {
		panic(err)
	}
	return ip
}

func cidrMustParse(s string) *CIDR {
	cidr, err := CIDRFromString(s)
	Expect(err).NotTo(HaveOccurred())
	return cidr
}

func bytePtr(b byte) *byte {
	return &b
}

func emptySubnetFromCidr(mainCidr string) *Subnet {
	cidr := cidrMustParse(mainCidr)
	return &Subnet{
		Spec: SubnetSpec{
			CIDR: cidr,
		},
		Status: SubnetStatus{
			Vacant:   []CIDR{},
			Reserved: cidr,
		},
	}
}

func subnetFromCidrs(mainCidr string, cidrStrings ...string) *Subnet {
	cidrs := make([]CIDR, len(cidrStrings))
	if len(cidrStrings) == 0 {
		cidrs = append(cidrs, *cidrMustParse(mainCidr))
	} else {
		for i, cidrString := range cidrStrings {
			cidrs[i] = *cidrMustParse(cidrString)
		}
	}

	cidr := cidrMustParse(mainCidr)
	return &Subnet{
		Spec: SubnetSpec{
			CIDR: cidr,
		},
		Status: SubnetStatus{
			Vacant:   cidrs,
			Reserved: cidr,
		},
	}
}

func networkFromCidrs(cidrStrings ...string) *Network {
	v4Cidrs := make([]CIDR, 0)
	v6Cidrs := make([]CIDR, 0)
	for _, cidrString := range cidrStrings {
		cidr := *cidrMustParse(cidrString)
		if cidr.IsIPv4() {
			v4Cidrs = append(v4Cidrs, cidr)
		} else {
			v6Cidrs = append(v6Cidrs, cidr)
		}
	}

	nw := &Network{
		Status: NetworkStatus{
			IPv4Ranges: v4Cidrs,
			IPv6Ranges: v6Cidrs,
		},
	}

	return nw
}
