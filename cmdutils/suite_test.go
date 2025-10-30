// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package cmdutils

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sSchema "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/ironcore-dev/controller-utils/modutils"
	ipamv1alphav1 "github.com/ironcore-dev/ipam/api/ipam/v1alpha1"
	//+kubebuilder:scaffold:imports
)

const (
	pollingInterval      = 50 * time.Millisecond
	eventuallyTimeout    = 3 * time.Second
	consistentlyDuration = 1 * time.Second
)

var (
	clients Clients
)

func TestBootctl(t *testing.T) {
	SetDefaultConsistentlyPollingInterval(pollingInterval)
	SetDefaultEventuallyPollingInterval(pollingInterval)
	SetDefaultEventuallyTimeout(eventuallyTimeout)
	SetDefaultConsistentlyDuration(consistentlyDuration)
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	sourceEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases"),
			filepath.Join(modutils.Dir("github.com/ironcore-dev/ipam", "config", "crd", "bases")),
		},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "bin", "k8s",
			fmt.Sprintf("1.34.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	sourceCfg, err := sourceEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(sourceCfg).NotTo(BeNil())

	DeferCleanup(sourceEnv.Stop)

	Expect(ipamv1alphav1.AddToScheme(k8sSchema.Scheme)).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	clients.Source, err = client.New(sourceCfg, client.Options{Scheme: k8sSchema.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(clients.Source).NotTo(BeNil())

	targetEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases"),
			filepath.Join(modutils.Dir("github.com/ironcore-dev/ipam", "config", "crd", "bases")),
		},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "bin", "k8s",
			fmt.Sprintf("1.34.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	// cfg is defined in this file globally.
	targetCfg, err := targetEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(targetCfg).NotTo(BeNil())

	DeferCleanup(targetEnv.Stop)

	clients.Target, err = client.New(targetCfg, client.Options{Scheme: k8sSchema.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(clients.Target).NotTo(BeNil())
})
