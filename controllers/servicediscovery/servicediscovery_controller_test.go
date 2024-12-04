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

package servicediscovery_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/names"
	submariner_v1 "github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/servicediscovery"
	opnames "github.com/submariner-io/submariner-operator/pkg/names"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Service discovery controller", func() {
	Context("Reconciliation", testReconciliation)
	Context("Deletion", func() {
		Context("Coredns cleanup", testCoreDNSCleanup)
		Context("Deployment cleanup", testDeploymentUninstall)
	})
})

func testReconciliation() {
	t := newTestDriver()

	It("should add a finalizer to the ServiceDiscovery resource", func(ctx SpecContext) {
		_, _ = t.DoReconcile(ctx)
		t.awaitFinalizer()
	})

	When("the openshift DNS config exists", func() {
		Context("and the lighthouse config isn't present", func() {
			BeforeEach(func() {
				t.InitScopedClientObjs = append(t.InitScopedClientObjs, newDNSService(clusterIP))
				t.InitGeneralClientObjs = append(t.InitGeneralClientObjs, newDNSConfig(""))
			})

			It("should add it", func(ctx SpecContext) {
				t.AssertReconcileSuccess(ctx)

				assertDNSConfigServers(t.assertDNSConfig(ctx), newDNSConfig(clusterIP))
			})
		})

		Context("and the lighthouse config is present and the lighthouse DNS service IP is updated", func() {
			updatedClusterIP := "10.10.10.11"

			BeforeEach(func() {
				t.InitScopedClientObjs = append(t.InitScopedClientObjs, newDNSService(updatedClusterIP))
				t.InitGeneralClientObjs = append(t.InitGeneralClientObjs, newDNSConfig(clusterIP))
			})

			It("should update the lighthouse config", func(ctx SpecContext) {
				t.AssertReconcileSuccess(ctx)

				assertDNSConfigServers(t.assertDNSConfig(ctx), newDNSConfig(updatedClusterIP))
			})
		})

		Context("and the lighthouse DNS service doesn't exist", func() {
			BeforeEach(func() {
				t.InitGeneralClientObjs = append(t.InitGeneralClientObjs, newDNSConfig(""))
			})

			It("should create the service and add the lighthouse config", func(ctx SpecContext) {
				t.AssertReconcileError(ctx)

				t.setLighthouseCoreDNSServiceIP(ctx)

				t.AssertReconcileSuccess(ctx)

				assertDNSConfigServers(t.assertDNSConfig(ctx), newDNSConfig(clusterIP))
			})
		})
	})

	When("the coredns ConfigMap exists", func() {
		Context("and the lighthouse config isn't present", func() {
			BeforeEach(func() {
				t.InitScopedClientObjs = append(t.InitScopedClientObjs, newDNSService(clusterIP))
				t.InitGeneralClientObjs = append(t.InitGeneralClientObjs, newCoreDNSConfigMap(coreDNSCorefileData("")))
			})

			It("should add it", func(ctx SpecContext) {
				t.AssertReconcileSuccess(ctx)

				Expect(strings.TrimSpace(t.assertCoreDNSConfigMap(ctx).Data[servicediscovery.Corefile])).To(Equal(coreDNSCorefileData(clusterIP)))
			})
		})

		Context("and the lighthouse config is present and the lighthouse DNS service IP is updated", func() {
			updatedClusterIP := "10.10.10.11"

			BeforeEach(func() {
				t.InitScopedClientObjs = append(t.InitScopedClientObjs, newDNSService(updatedClusterIP))
				t.InitGeneralClientObjs = append(t.InitGeneralClientObjs, newCoreDNSConfigMap(coreDNSCorefileData(clusterIP)))
			})

			It("should update the lighthouse config", func(ctx SpecContext) {
				t.AssertReconcileSuccess(ctx)

				Expect(strings.TrimSpace(t.assertCoreDNSConfigMap(ctx).Data[servicediscovery.Corefile])).To(
					Equal(coreDNSCorefileData(updatedClusterIP)))
			})
		})

		Context("and the lighthouse DNS service doesn't exist", func() {
			BeforeEach(func() {
				t.InitGeneralClientObjs = append(t.InitGeneralClientObjs, newCoreDNSConfigMap(coreDNSCorefileData("")))
			})

			It("should create the service and add the lighthouse config", func(ctx SpecContext) {
				t.AssertReconcileError(ctx)

				t.setLighthouseCoreDNSServiceIP(ctx)

				t.AssertReconcileSuccess(ctx)

				Expect(strings.TrimSpace(t.assertCoreDNSConfigMap(ctx).Data[servicediscovery.Corefile])).
					To(Equal(coreDNSCorefileData(clusterIP)))
			})
		})
	})

	When("a custom coredns config is specified", func() {
		BeforeEach(func() {
			t.serviceDiscovery.Spec.CoreDNSCustomConfig = &submariner_v1.CoreDNSCustomConfig{
				ConfigMapName: "custom-config",
				Namespace:     "custom-config-ns",
			}
		})

		Context("and the custom coredns ConfigMap already exists", func() {
			BeforeEach(func() {
				t.InitScopedClientObjs = append(t.InitScopedClientObjs, newDNSService(clusterIP), &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      t.serviceDiscovery.Spec.CoreDNSCustomConfig.ConfigMapName,
						Namespace: t.serviceDiscovery.Spec.CoreDNSCustomConfig.Namespace,
					},
					Data: map[string]string{
						"lighthouse.server": strings.ReplaceAll(coreDNSConfigFormat, "$IP", "1.2.3.4"),
					},
				})
			})

			It("should update it", func(ctx SpecContext) {
				t.AssertReconcileSuccess(ctx)

				Expect(strings.TrimSpace(t.assertConfigMap(ctx, t.serviceDiscovery.Spec.CoreDNSCustomConfig.ConfigMapName,
					t.serviceDiscovery.Spec.CoreDNSCustomConfig.Namespace).Data["lighthouse.server"])).To(Equal(
					strings.ReplaceAll(lighthouseDNSConfigFormat, "$IP", clusterIP)))
			})
		})

		Context("and the custom coredns ConfigMap doesn't exist", func() {
			BeforeEach(func() {
				t.InitScopedClientObjs = append(t.InitScopedClientObjs, newDNSService(clusterIP))
			})

			It("should create it", func(ctx SpecContext) {
				t.AssertReconcileSuccess(ctx)

				Expect(strings.TrimSpace(t.assertConfigMap(ctx, t.serviceDiscovery.Spec.CoreDNSCustomConfig.ConfigMapName,
					t.serviceDiscovery.Spec.CoreDNSCustomConfig.Namespace).Data["lighthouse.server"])).To(Equal(
					strings.ReplaceAll(lighthouseDNSConfigFormat, "$IP", clusterIP)))
			})
		})

		Context("and the lighthouse DNS service doesn't exist", func() {
			It("should create the service and the custom coredns ConfigMap", func(ctx SpecContext) {
				t.AssertReconcileError(ctx)

				t.setLighthouseCoreDNSServiceIP(ctx)

				t.AssertReconcileSuccess(ctx)

				Expect(strings.TrimSpace(t.assertConfigMap(ctx, t.serviceDiscovery.Spec.CoreDNSCustomConfig.ConfigMapName,
					t.serviceDiscovery.Spec.CoreDNSCustomConfig.Namespace).Data["lighthouse.server"])).To(Equal(
					strings.ReplaceAll(lighthouseDNSConfigFormat, "$IP", clusterIP)))
			})
		})
	})
}

func testCoreDNSCleanup() {
	t := newTestDriver()

	BeforeEach(func() {
		t.serviceDiscovery.SetFinalizers([]string{opnames.CleanupFinalizer})

		now := metav1.Now()
		t.serviceDiscovery.SetDeletionTimestamp(&now)
	})

	JustBeforeEach(func(ctx SpecContext) {
		deployment := t.NewDeployment(opnames.AppendUninstall(names.ServiceDiscoveryComponent))

		var one int32 = 1
		deployment.Spec.Replicas = &one

		Expect(t.ScopedClient.Create(ctx, deployment)).To(Succeed())
		t.UpdateDeploymentToReady(ctx, deployment)

		t.AssertReconcileSuccess(ctx)
	})

	When("the coredns ConfigMap exists", func() {
		BeforeEach(func() {
			t.InitGeneralClientObjs = append(t.InitGeneralClientObjs, newCoreDNSConfigMap(coreDNSCorefileData(clusterIP)))
		})

		It("should remove the lighthouse config section", func(ctx SpecContext) {
			Expect(strings.TrimSpace(t.assertCoreDNSConfigMap(ctx).Data[servicediscovery.Corefile])).To(
				Equal(coreDNSCorefileData("")))
		})

		t.testServiceDiscoveryDeleted()
	})

	When("the openshift DNS config exists", func() {
		BeforeEach(func() {
			t.InitGeneralClientObjs = append(t.InitGeneralClientObjs, newDNSConfig(clusterIP))
		})

		It("should remove the lighthouse config", func(ctx SpecContext) {
			assertDNSConfigServers(t.assertDNSConfig(ctx), newDNSConfig(""))
		})

		t.testServiceDiscoveryDeleted()
	})

	When("a custom coredns config is specified", func() {
		BeforeEach(func() {
			t.serviceDiscovery.Spec.CoreDNSCustomConfig = &submariner_v1.CoreDNSCustomConfig{
				ConfigMapName: "custom-config",
				Namespace:     "custom-config-ns",
			}
		})

		Context("and the custom coredns ConfigMap exists", func() {
			BeforeEach(func() {
				t.InitGeneralClientObjs = append(t.InitGeneralClientObjs, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      t.serviceDiscovery.Spec.CoreDNSCustomConfig.ConfigMapName,
						Namespace: t.serviceDiscovery.Spec.CoreDNSCustomConfig.Namespace,
					},
					Data: map[string]string{
						"lighthouse.server": strings.ReplaceAll(coreDNSConfigFormat, "$IP", clusterIP),
					},
				})
			})

			It("should remove the lighthouse config section", func(ctx SpecContext) {
				Expect(t.assertConfigMap(ctx, t.serviceDiscovery.Spec.CoreDNSCustomConfig.ConfigMapName,
					t.serviceDiscovery.Spec.CoreDNSCustomConfig.Namespace).Data).ToNot(HaveKey("lighthouse.server"))
			})

			t.testServiceDiscoveryDeleted()
		})

		Context("and the custom coredns ConfigMap doesn't exist", func() {
			t.testServiceDiscoveryDeleted()
		})
	})
}

func testDeploymentUninstall() {
	t := newTestDriver()

	BeforeEach(func() {
		t.serviceDiscovery.SetFinalizers([]string{opnames.CleanupFinalizer})

		now := metav1.Now()
		t.serviceDiscovery.SetDeletionTimestamp(&now)
	})

	Context("", func() {
		BeforeEach(func() {
			t.InitScopedClientObjs = append(t.InitScopedClientObjs,
				t.NewDeployment(names.ServiceDiscoveryComponent))
		})

		It("should run a Deployment to uninstall the service discovery component", func(ctx SpecContext) {
			t.AssertReconcileRequeue(ctx)

			t.AssertNoDeployment(ctx, names.ServiceDiscoveryComponent)

			t.UpdateDeploymentToReady(ctx, t.assertUninstallServiceDiscoveryDeployment(ctx))

			t.awaitFinalizer()

			t.AssertReconcileSuccess(ctx)

			t.AssertNoDeployment(ctx, opnames.AppendUninstall(names.ServiceDiscoveryComponent))

			t.awaitServiceDiscoveryDeleted()

			t.AssertReconcileSuccess(ctx)
			t.AssertNoDeployment(ctx, opnames.AppendUninstall(names.ServiceDiscoveryComponent))
		})
	})

	When("the version of the deleting ServiceDiscovery instance does not support uninstall", func() {
		BeforeEach(func() {
			t.serviceDiscovery.Spec.Version = "0.11.1"

			t.InitScopedClientObjs = append(t.InitScopedClientObjs, t.NewDeployment(names.ServiceDiscoveryComponent))
		})

		It("should not perform uninstall", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)

			_, err := t.GetDeployment(ctx, names.ServiceDiscoveryComponent)
			Expect(err).To(Succeed())

			t.AssertNoDeployment(ctx, opnames.AppendUninstall(names.ServiceDiscoveryComponent))

			t.awaitServiceDiscoveryDeleted()
		})
	})
}
