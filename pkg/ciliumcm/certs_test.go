/*
SPDX-License-Identifier: Apache-2.0

Copyright Contributors to the Submariner project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ciliumcm_test

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/ciliumcm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCiliumCM(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cilium CM Suite")
}

var _ = Describe("GenerateBundle", func() {
	It("should produce CA, server, and client PEMs usable together", func() {
		bundle, err := ciliumcm.GenerateBundle(24 * time.Hour)
		Expect(err).NotTo(HaveOccurred())
		Expect(bundle.CACert).NotTo(BeEmpty())
		Expect(bundle.CAKey).NotTo(BeEmpty())
		Expect(bundle.ServerCert).NotTo(BeEmpty())
		Expect(bundle.ClientCert).NotTo(BeEmpty())

		cert, err := tls.X509KeyPair(bundle.ServerCert, bundle.ServerKey)
		Expect(err).NotTo(HaveOccurred())
		Expect(cert.Certificate).NotTo(BeEmpty())

		pool := x509.NewCertPool()
		Expect(pool.AppendCertsFromPEM(bundle.CACert)).To(BeTrue())

		data := bundle.SecretData()
		Expect(data).To(HaveKey(ciliumcm.CACertKey))
		Expect(data).To(HaveKey(ciliumcm.CAKeyKey))
		Expect(data).To(HaveKey(ciliumcm.TLSCertKey))
		Expect(data).To(HaveKey(ciliumcm.ClientKeyKey))
	})

	It("should render ClusterMesh peer config YAML", func() {
		cfg := string(ciliumcm.PeerConfigYAML("submariner", ciliumcm.DefaultListenURL))
		Expect(cfg).To(ContainSubstring("endpoints:"))
		Expect(cfg).To(ContainSubstring(ciliumcm.DefaultListenURL))
		Expect(cfg).To(ContainSubstring("submariner.etcd-client-ca.crt"))
	})

	It("should list only Submariner-owned peer Secret keys", func() {
		Expect(ciliumcm.PeerSecretKeys("submariner")).To(Equal([]string{
			"submariner",
			"submariner.etcd-client-ca.crt",
			"submariner.etcd-client.crt",
			"submariner.etcd-client.key",
		}))
	})
})

var _ = Describe("AssessSecretData", func() {
	It("should return RenewNone for a fresh complete bundle", func() {
		bundle, err := ciliumcm.GenerateBundle(365 * 24 * time.Hour)
		Expect(err).NotTo(HaveOccurred())

		action, reason := ciliumcm.AssessSecretData(bundle.SecretData(), time.Now(), ciliumcm.DefaultRenewBefore)
		Expect(action).To(Equal(ciliumcm.RenewNone), reason)
	})

	It("should return RenewFull when required keys are missing", func() {
		action, reason := ciliumcm.AssessSecretData(map[string][]byte{
			ciliumcm.CACertKey: []byte("x"),
		}, time.Now(), ciliumcm.DefaultRenewBefore)
		Expect(action).To(Equal(ciliumcm.RenewFull))
		Expect(reason).To(ContainSubstring("missing key"))
	})

	It("should return RenewLeaves when leaf is in renew window and ca.key is present", func() {
		bundle, err := ciliumcm.GenerateBundle(48 * time.Hour)
		Expect(err).NotTo(HaveOccurred())

		// renewBefore > remaining lifetime → leaf renew
		action, reason := ciliumcm.AssessSecretData(bundle.SecretData(), time.Now(), 7*24*time.Hour)
		Expect(action).To(Equal(ciliumcm.RenewLeaves), reason)
	})

	It("should return RenewFull when leaf needs renew but ca.key is absent", func() {
		bundle, err := ciliumcm.GenerateBundle(48 * time.Hour)
		Expect(err).NotTo(HaveOccurred())

		data := bundle.SecretData()
		delete(data, ciliumcm.CAKeyKey)

		action, reason := ciliumcm.AssessSecretData(data, time.Now(), 7*24*time.Hour)
		Expect(action).To(Equal(ciliumcm.RenewFull), reason)
		Expect(reason).To(ContainSubstring("no ca.key"))
	})

	It("should tolerate legacy secrets without ca.key while still valid", func() {
		bundle, err := ciliumcm.GenerateBundle(365 * 24 * time.Hour)
		Expect(err).NotTo(HaveOccurred())

		data := bundle.SecretData()
		delete(data, ciliumcm.CAKeyKey)

		action, reason := ciliumcm.AssessSecretData(data, time.Now(), ciliumcm.DefaultRenewBefore)
		Expect(action).To(Equal(ciliumcm.RenewNone), reason)
		Expect(reason).To(ContainSubstring("ca.key absent"))
	})
})

var _ = Describe("ReissueLeaves", func() {
	It("should keep the CA and re-issue server/client certs", func() {
		bundle, err := ciliumcm.GenerateBundle(24 * time.Hour)
		Expect(err).NotTo(HaveOccurred())

		renewed, err := ciliumcm.ReissueLeaves(bundle.CACert, bundle.CAKey, 365*24*time.Hour)
		Expect(err).NotTo(HaveOccurred())

		Expect(string(renewed.CACert)).To(Equal(string(bundle.CACert)))
		Expect(string(renewed.CAKey)).To(Equal(string(bundle.CAKey)))
		Expect(string(renewed.ServerCert)).NotTo(Equal(string(bundle.ServerCert)))
		Expect(string(renewed.ClientCert)).NotTo(Equal(string(bundle.ClientCert)))

		action, reason := ciliumcm.AssessSecretData(renewed.SecretData(), time.Now(), ciliumcm.DefaultRenewBefore)
		Expect(action).To(Equal(ciliumcm.RenewNone), reason)
	})
})

var _ = Describe("FindUniqueCiliumConfigNamespace", func() {
	It("should return the namespace when exactly one cilium-config exists", func(ctx SpecContext) {
		client := fake.NewSimpleClientset(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: ciliumcm.CiliumConfigMapName, Namespace: "cilium"},
		})

		ns, err := ciliumcm.FindUniqueCiliumConfigNamespace(ctx, client)
		Expect(err).NotTo(HaveOccurred())
		Expect(ns).To(Equal("cilium"))
	})

	It("should return empty when none or multiple exist", func(ctx SpecContext) {
		client := fake.NewSimpleClientset()
		ns, err := ciliumcm.FindUniqueCiliumConfigNamespace(ctx, client)
		Expect(err).NotTo(HaveOccurred())
		Expect(ns).To(BeEmpty())

		client = fake.NewSimpleClientset(
			&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: ciliumcm.CiliumConfigMapName, Namespace: "a"}},
			&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: ciliumcm.CiliumConfigMapName, Namespace: "b"}},
		)
		ns, err = ciliumcm.FindUniqueCiliumConfigNamespace(ctx, client)
		Expect(err).NotTo(HaveOccurred())
		Expect(ns).To(BeEmpty())
	})
})
