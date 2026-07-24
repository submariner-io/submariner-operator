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

package submariner_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/names"
	"github.com/submariner-io/submariner-operator/internal/controllers/test"
	"github.com/submariner-io/submariner-operator/pkg/ciliumcm"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	opnames "github.com/submariner-io/submariner-operator/pkg/names"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const legacySubmarinerVersion = "0.11.1"

var _ = Describe("Cilium CM publisher wiring", func() {
	t := newTestDriver()

	BeforeEach(func() {
		t.clusterNetwork = &network.ClusterNetwork{
			NetworkPlugin: "cilium",
			ServiceCIDRs:  []string{"100.94.0.0/16"},
			PodCIDRs:      []string{"10.244.0.0/16"},
		}
	})

	When("cilium-config has a valid cluster-id", func() {
		BeforeEach(func() {
			t.InitGeneralClientObjs = []controllerClient.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ciliumcm.CiliumConfigMapName,
						Namespace: metav1.NamespaceSystem,
					},
					Data: map[string]string{
						"cluster-id":   "1",
						"cluster-name": "test-cluster",
					},
				},
			}
		})

		It("should create TLS Secret, merge cilium-clustermesh, and mount TLS on route-agent", func(ctx SpecContext) {
			result, err := t.DoReconcile(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(ciliumcm.CertCheckRequeue))

			tls := &corev1.Secret{}
			Expect(t.ScopedClient.Get(ctx, types.NamespacedName{
				Name: ciliumcm.TLSSecretName, Namespace: submarinerNamespace,
			}, tls)).To(Succeed())
			Expect(tls.Data).To(HaveKey(ciliumcm.CACertKey))
			Expect(tls.Data).To(HaveKey(ciliumcm.CAKeyKey))
			Expect(tls.Data).To(HaveKey(ciliumcm.TLSCertKey))
			Expect(tls.Data).To(HaveKey(ciliumcm.ClientCertKey))

			mesh := &corev1.Secret{}
			Expect(t.GeneralClient.Get(ctx, types.NamespacedName{
				Name: ciliumcm.ClusterMeshSecretName, Namespace: metav1.NamespaceSystem,
			}, mesh)).To(Succeed())
			Expect(mesh.Data).To(HaveKey(ciliumcm.DefaultRemoteName))
			Expect(mesh.Data).To(HaveKey(ciliumcm.DefaultRemoteName + ".etcd-client.crt"))

			daemonSet := t.AssertDaemonSet(ctx, names.RouteAgentComponent)

			foundVol := false

			for i := range daemonSet.Spec.Template.Spec.Volumes {
				if daemonSet.Spec.Template.Spec.Volumes[i].Name == ciliumcm.VolumeName {
					foundVol = true
					Expect(daemonSet.Spec.Template.Spec.Volumes[i].Secret).NotTo(BeNil())
					Expect(daemonSet.Spec.Template.Spec.Volumes[i].Secret.SecretName).To(Equal(ciliumcm.TLSSecretName))
				}
			}
			Expect(foundVol).To(BeTrue())

			envMap := test.EnvMapFrom(daemonSet)
			Expect(envMap).To(HaveKeyWithValue("SUBMARINER_CILIUM_CM_LISTEN_URL", ciliumcm.DefaultListenURL))
			Expect(envMap).To(HaveKeyWithValue("SUBMARINER_NETWORKPLUGIN", "cilium"))
		})

		It("should renew leaves without changing CA", func(ctx SpecContext) {
			_, err := t.DoReconcile(ctx)
			Expect(err).NotTo(HaveOccurred())

			tls := &corev1.Secret{}
			Expect(t.ScopedClient.Get(ctx, types.NamespacedName{
				Name: ciliumcm.TLSSecretName, Namespace: submarinerNamespace,
			}, tls)).To(Succeed())

			originalCA := append([]byte(nil), tls.Data[ciliumcm.CACertKey]...)
			originalServer := append([]byte(nil), tls.Data[ciliumcm.TLSCertKey]...)

			// Force leaf renew on next reconcile by installing a short-lived leaf set.
			short, err := ciliumcm.ReissueLeaves(tls.Data[ciliumcm.CACertKey], tls.Data[ciliumcm.CAKeyKey], time.Hour)
			Expect(err).NotTo(HaveOccurred())

			tls.Data = short.SecretData()
			Expect(t.ScopedClient.Update(ctx, tls)).To(Succeed())

			_, err = t.DoReconcile(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(t.ScopedClient.Get(ctx, types.NamespacedName{
				Name: ciliumcm.TLSSecretName, Namespace: submarinerNamespace,
			}, tls)).To(Succeed())
			Expect(string(tls.Data[ciliumcm.CACertKey])).To(Equal(string(originalCA)))
			Expect(string(tls.Data[ciliumcm.TLSCertKey])).NotTo(Equal(string(originalServer)))

			mesh := &corev1.Secret{}
			Expect(t.GeneralClient.Get(ctx, types.NamespacedName{
				Name: ciliumcm.ClusterMeshSecretName, Namespace: metav1.NamespaceSystem,
			}, mesh)).To(Succeed())
			Expect(string(mesh.Data[ciliumcm.DefaultRemoteName+".etcd-client.crt"])).
				To(Equal(string(tls.Data[ciliumcm.ClientCertKey])))
		})

		It("should keep a valid existing Secret unchanged across reconciles", func(ctx SpecContext) {
			_, err := t.DoReconcile(ctx)
			Expect(err).NotTo(HaveOccurred())

			tls := &corev1.Secret{}
			Expect(t.ScopedClient.Get(ctx, types.NamespacedName{
				Name: ciliumcm.TLSSecretName, Namespace: submarinerNamespace,
			}, tls)).To(Succeed())
			serverBefore := append([]byte(nil), tls.Data[ciliumcm.TLSCertKey]...)
			rvBefore := tls.ResourceVersion

			_, err = t.DoReconcile(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(t.ScopedClient.Get(ctx, types.NamespacedName{
				Name: ciliumcm.TLSSecretName, Namespace: submarinerNamespace,
			}, tls)).To(Succeed())
			Expect(string(tls.Data[ciliumcm.TLSCertKey])).To(Equal(string(serverBefore)))
			Expect(tls.ResourceVersion).To(Equal(rvBefore))
		})

		It("should fully regenerate TLS when required keys are missing", func(ctx SpecContext) {
			_, err := t.DoReconcile(ctx)
			Expect(err).NotTo(HaveOccurred())

			tls := &corev1.Secret{}
			Expect(t.ScopedClient.Get(ctx, types.NamespacedName{
				Name: ciliumcm.TLSSecretName, Namespace: submarinerNamespace,
			}, tls)).To(Succeed())

			originalCA := append([]byte(nil), tls.Data[ciliumcm.CACertKey]...)
			delete(tls.Data, ciliumcm.ClientCertKey)
			Expect(t.ScopedClient.Update(ctx, tls)).To(Succeed())

			_, err = t.DoReconcile(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(t.ScopedClient.Get(ctx, types.NamespacedName{
				Name: ciliumcm.TLSSecretName, Namespace: submarinerNamespace,
			}, tls)).To(Succeed())
			Expect(tls.Data).To(HaveKey(ciliumcm.ClientCertKey))
			Expect(string(tls.Data[ciliumcm.CACertKey])).NotTo(Equal(string(originalCA)))

			mesh := &corev1.Secret{}
			Expect(t.GeneralClient.Get(ctx, types.NamespacedName{
				Name: ciliumcm.ClusterMeshSecretName, Namespace: metav1.NamespaceSystem,
			}, mesh)).To(Succeed())
			Expect(string(mesh.Data[ciliumcm.DefaultRemoteName+".etcd-client.crt"])).
				To(Equal(string(tls.Data[ciliumcm.ClientCertKey])))
		})

		It("should merge Submariner peer into an existing shared cilium-clustermesh Secret", func(ctx SpecContext) {
			Expect(t.GeneralClient.Create(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ciliumcm.ClusterMeshSecretName,
					Namespace: metav1.NamespaceSystem,
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"other-cluster":                 []byte("endpoints:\n- https://10.0.0.1:2379\n"),
					"other-cluster.etcd-client.crt": []byte("other-crt"),
					ciliumcm.DefaultRemoteName:      []byte("stale-config"),
				},
			})).To(Succeed())

			_, err := t.DoReconcile(ctx)
			Expect(err).NotTo(HaveOccurred())

			tls := &corev1.Secret{}
			Expect(t.ScopedClient.Get(ctx, types.NamespacedName{
				Name: ciliumcm.TLSSecretName, Namespace: submarinerNamespace,
			}, tls)).To(Succeed())

			mesh := &corev1.Secret{}
			Expect(t.GeneralClient.Get(ctx, types.NamespacedName{
				Name: ciliumcm.ClusterMeshSecretName, Namespace: metav1.NamespaceSystem,
			}, mesh)).To(Succeed())
			Expect(mesh.Data).To(HaveKey("other-cluster"))
			Expect(mesh.Data).To(HaveKeyWithValue("other-cluster.etcd-client.crt", []byte("other-crt")))
			Expect(string(mesh.Data[ciliumcm.DefaultRemoteName])).NotTo(Equal("stale-config"))
			Expect(string(mesh.Data[ciliumcm.DefaultRemoteName+".etcd-client.crt"])).
				To(Equal(string(tls.Data[ciliumcm.ClientCertKey])))
		})
	})

	assertTLSWithoutClusterMeshMerge := func(ctx SpecContext) {
		_, err := t.DoReconcile(ctx)
		Expect(err).NotTo(HaveOccurred())

		tls := &corev1.Secret{}
		Expect(t.ScopedClient.Get(ctx, types.NamespacedName{
			Name: ciliumcm.TLSSecretName, Namespace: submarinerNamespace,
		}, tls)).To(Succeed())

		mesh := &corev1.Secret{}
		err = t.GeneralClient.Get(ctx, types.NamespacedName{
			Name: ciliumcm.ClusterMeshSecretName, Namespace: metav1.NamespaceSystem,
		}, mesh)
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	}

	When("cilium-config cluster-id is 0", func() {
		BeforeEach(func() {
			t.InitGeneralClientObjs = []controllerClient.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ciliumcm.CiliumConfigMapName,
						Namespace: metav1.NamespaceSystem,
					},
					Data: map[string]string{
						"cluster-id":   "0",
						"cluster-name": "test-cluster",
					},
				},
			}
		})

		It("should create TLS Secret but not merge cilium-clustermesh", assertTLSWithoutClusterMeshMerge)
	})

	When("cilium-config cluster-name is default", func() {
		BeforeEach(func() {
			t.InitGeneralClientObjs = []controllerClient.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ciliumcm.CiliumConfigMapName,
						Namespace: metav1.NamespaceSystem,
					},
					Data: map[string]string{
						"cluster-id":   "1",
						"cluster-name": "default",
					},
				},
			}
		})

		It("should create TLS Secret but not merge cilium-clustermesh", assertTLSWithoutClusterMeshMerge)
	})

	When("NetworkPlugin is not cilium and a leftover Submariner peer exists", func() {
		BeforeEach(func() {
			t.clusterNetwork.NetworkPlugin = "ovn-kubernetes"
			t.InitGeneralClientObjs = []controllerClient.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ciliumcm.ClusterMeshSecretName,
						Namespace: metav1.NamespaceSystem,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						ciliumcm.DefaultRemoteName:                         []byte("endpoints:\n- https://127.0.0.1:12379\n"),
						ciliumcm.DefaultRemoteName + ".etcd-client-ca.crt": []byte("ca"),
						ciliumcm.DefaultRemoteName + ".etcd-client.crt":    []byte("crt"),
						ciliumcm.DefaultRemoteName + ".etcd-client.key":    []byte("key"),
						"other-cluster":                 []byte("endpoints:\n- https://10.0.0.1:2379\n"),
						"other-cluster.etcd-client.crt": []byte("other-crt"),
					},
				},
			}
		})

		It("should remove only Submariner peer keys", func(ctx SpecContext) {
			_, err := t.DoReconcile(ctx)
			Expect(err).NotTo(HaveOccurred())

			mesh := &corev1.Secret{}
			Expect(t.GeneralClient.Get(ctx, types.NamespacedName{
				Name: ciliumcm.ClusterMeshSecretName, Namespace: metav1.NamespaceSystem,
			}, mesh)).To(Succeed())
			Expect(mesh.Data).NotTo(HaveKey(ciliumcm.DefaultRemoteName))
			Expect(mesh.Data).NotTo(HaveKey(ciliumcm.DefaultRemoteName + ".etcd-client.crt"))
			Expect(mesh.Data).To(HaveKey("other-cluster"))
			Expect(mesh.Data).To(HaveKeyWithValue("other-cluster.etcd-client.crt", []byte("other-crt")))
		})
	})

	When("Submariner is being deleted with a shared cilium-clustermesh Secret", func() {
		BeforeEach(func() {
			t.submariner.SetFinalizers([]string{opnames.CleanupFinalizer})

			ts := metav1.Now()
			t.submariner.SetDeletionTimestamp(&ts)
			// Skip uninstall DaemonSets so the test focuses on peer cleanup.
			t.submariner.Spec.Version = legacySubmarinerVersion
			t.InitGeneralClientObjs = []controllerClient.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ciliumcm.ClusterMeshSecretName,
						Namespace: metav1.NamespaceSystem,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						ciliumcm.DefaultRemoteName:                         []byte("endpoints:\n- https://127.0.0.1:12379\n"),
						ciliumcm.DefaultRemoteName + ".etcd-client-ca.crt": []byte("ca"),
						ciliumcm.DefaultRemoteName + ".etcd-client.crt":    []byte("crt"),
						ciliumcm.DefaultRemoteName + ".etcd-client.key":    []byte("key"),
						"other-cluster":                 []byte("endpoints:\n- https://10.0.0.1:2379\n"),
						"other-cluster.etcd-client.crt": []byte("other-crt"),
					},
				},
			}
		})

		It("should remove only Submariner keys and preserve other peers", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)

			mesh := &corev1.Secret{}
			Expect(t.GeneralClient.Get(ctx, types.NamespacedName{
				Name: ciliumcm.ClusterMeshSecretName, Namespace: metav1.NamespaceSystem,
			}, mesh)).To(Succeed())
			Expect(mesh.Data).NotTo(HaveKey(ciliumcm.DefaultRemoteName))
			Expect(mesh.Data).NotTo(HaveKey(ciliumcm.DefaultRemoteName + ".etcd-client-ca.crt"))
			Expect(mesh.Data).NotTo(HaveKey(ciliumcm.DefaultRemoteName + ".etcd-client.crt"))
			Expect(mesh.Data).NotTo(HaveKey(ciliumcm.DefaultRemoteName + ".etcd-client.key"))
			Expect(mesh.Data).To(HaveKey("other-cluster"))
			Expect(mesh.Data).To(HaveKeyWithValue("other-cluster.etcd-client.crt", []byte("other-crt")))
		})
	})

	When("Submariner is being deleted and cilium-clustermesh has only Submariner keys", func() {
		BeforeEach(func() {
			t.submariner.SetFinalizers([]string{opnames.CleanupFinalizer})

			ts := metav1.Now()
			t.submariner.SetDeletionTimestamp(&ts)
			t.submariner.Spec.Version = legacySubmarinerVersion
			t.InitGeneralClientObjs = []controllerClient.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ciliumcm.ClusterMeshSecretName,
						Namespace: metav1.NamespaceSystem,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						ciliumcm.DefaultRemoteName:                         []byte("endpoints:\n- https://127.0.0.1:12379\n"),
						ciliumcm.DefaultRemoteName + ".etcd-client-ca.crt": []byte("ca"),
						ciliumcm.DefaultRemoteName + ".etcd-client.crt":    []byte("crt"),
						ciliumcm.DefaultRemoteName + ".etcd-client.key":    []byte("key"),
					},
				},
			}
		})

		It("should delete the empty cilium-clustermesh Secret", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)

			mesh := &corev1.Secret{}
			err := t.GeneralClient.Get(ctx, types.NamespacedName{
				Name: ciliumcm.ClusterMeshSecretName, Namespace: metav1.NamespaceSystem,
			}, mesh)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})
})
