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
	"context"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	submariner_v1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/constants"
	"github.com/submariner-io/submariner-operator/pkg/names"
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

	It("should add a finalizer to the ServiceDiscovery resource", func() {
		_, _ = t.DoReconcile()
		t.awaitFinalizer()
	})

	When("the openshift DNS config exists", func() {
		Context("and the lighthouse config isn't present", func() {
			BeforeEach(func() {
				t.InitClientObjs = append(t.InitClientObjs, newDNSConfig(""), newDNSService(clusterIP))
			})

			It("should add it", func() {
				t.AssertReconcileSuccess()

				assertDNSConfigServers(t.assertDNSConfig(), newDNSConfig(clusterIP))
			})
		})

		Context("and the lighthouse config is present and the lighthouse DNS service IP is updated", func() {
			updatedClusterIP := "10.10.10.11"

			BeforeEach(func() {
				t.InitClientObjs = append(t.InitClientObjs, newDNSConfig(clusterIP), newDNSService(updatedClusterIP))
			})

			It("should update the lighthouse config", func() {
				t.AssertReconcileSuccess()

				assertDNSConfigServers(t.assertDNSConfig(), newDNSConfig(updatedClusterIP))
			})
		})

		Context("and the lighthouse DNS service doesn't exist", func() {
			BeforeEach(func() {
				t.InitClientObjs = append(t.InitClientObjs, newDNSConfig(""))
			})

			It("should create the service and add the lighthouse config", func() {
				t.AssertReconcileError()

				t.setLighthouseCoreDNSServiceIP()

				t.AssertReconcileSuccess()

				assertDNSConfigServers(t.assertDNSConfig(), newDNSConfig(clusterIP))
			})
		})
	})

	When("the coredns ConfigMap exists", func() {
		Context("and the lighthouse config isn't present", func() {
			BeforeEach(func() {
				t.InitClientObjs = append(t.InitClientObjs, newDNSService(clusterIP))
				t.createConfigMap(newCoreDNSConfigMap(coreDNSCorefileData("")))
			})

			It("should add it", func() {
				t.AssertReconcileSuccess()

				Expect(strings.TrimSpace(t.assertCoreDNSConfigMap().Data["Corefile"])).To(Equal(coreDNSCorefileData(clusterIP)))
			})
		})

		Context("and the lighthouse config is present and the lighthouse DNS service IP is updated", func() {
			updatedClusterIP := "10.10.10.11"

			BeforeEach(func() {
				t.InitClientObjs = append(t.InitClientObjs, newDNSService(updatedClusterIP))
				t.createConfigMap(newCoreDNSConfigMap(coreDNSCorefileData(clusterIP)))
			})

			It("should update the lighthouse config", func() {
				t.AssertReconcileSuccess()

				Expect(strings.TrimSpace(t.assertCoreDNSConfigMap().Data["Corefile"])).To(Equal(coreDNSCorefileData(updatedClusterIP)))
			})
		})

		Context("and the lighthouse DNS service doesn't exist", func() {
			BeforeEach(func() {
				t.createConfigMap(newCoreDNSConfigMap(coreDNSCorefileData("")))
			})

			It("should create the service and add the lighthouse config", func() {
				t.AssertReconcileError()

				t.setLighthouseCoreDNSServiceIP()

				t.AssertReconcileSuccess()

				Expect(strings.TrimSpace(t.assertCoreDNSConfigMap().Data["Corefile"])).To(Equal(coreDNSCorefileData(clusterIP)))
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
				t.createConfigMap(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      t.serviceDiscovery.Spec.CoreDNSCustomConfig.ConfigMapName,
						Namespace: t.serviceDiscovery.Spec.CoreDNSCustomConfig.Namespace,
					},
					Data: map[string]string{
						"lighthouse.server": strings.ReplaceAll(coreDNSConfigFormat, "$IP", "1.2.3.4"),
					},
				})

				t.InitClientObjs = append(t.InitClientObjs, newDNSService(clusterIP))
			})

			It("should update it", func() {
				t.AssertReconcileSuccess()

				Expect(strings.TrimSpace(t.assertConfigMap(t.serviceDiscovery.Spec.CoreDNSCustomConfig.ConfigMapName,
					t.serviceDiscovery.Spec.CoreDNSCustomConfig.Namespace).Data["lighthouse.server"])).To(Equal(
					strings.ReplaceAll(lighthouseDNSConfigFormat, "$IP", clusterIP)))
			})
		})

		Context("and the custom coredns ConfigMap doesn't exist", func() {
			BeforeEach(func() {
				t.InitClientObjs = append(t.InitClientObjs, newDNSService(clusterIP))
			})

			It("should create it", func() {
				t.AssertReconcileSuccess()

				Expect(strings.TrimSpace(t.assertConfigMap(t.serviceDiscovery.Spec.CoreDNSCustomConfig.ConfigMapName,
					t.serviceDiscovery.Spec.CoreDNSCustomConfig.Namespace).Data["lighthouse.server"])).To(Equal(
					strings.ReplaceAll(lighthouseDNSConfigFormat, "$IP", clusterIP)))
			})
		})

		Context("and the lighthouse DNS service doesn't exist", func() {
			It("should create the service and the custom coredns ConfigMap", func() {
				t.AssertReconcileError()

				t.setLighthouseCoreDNSServiceIP()

				t.AssertReconcileSuccess()

				Expect(strings.TrimSpace(t.assertConfigMap(t.serviceDiscovery.Spec.CoreDNSCustomConfig.ConfigMapName,
					t.serviceDiscovery.Spec.CoreDNSCustomConfig.Namespace).Data["lighthouse.server"])).To(Equal(
					strings.ReplaceAll(lighthouseDNSConfigFormat, "$IP", clusterIP)))
			})
		})
	})
}

func testCoreDNSCleanup() {
	t := newTestDriver()

	BeforeEach(func() {
		t.serviceDiscovery.SetFinalizers([]string{constants.CleanupFinalizer})

		now := metav1.Now()
		t.serviceDiscovery.SetDeletionTimestamp(&now)
	})

	JustBeforeEach(func() {
		deployment := t.NewDeployment(names.AppendUninstall(names.ServiceDiscoveryComponent))

		var one int32 = 1
		deployment.Spec.Replicas = &one

		Expect(t.Client.Create(context.TODO(), deployment)).To(Succeed())
		t.UpdateDeploymentToReady(deployment)

		t.AssertReconcileSuccess()
	})

	When("the coredns ConfigMap exists", func() {
		BeforeEach(func() {
			t.createConfigMap(newCoreDNSConfigMap(coreDNSCorefileData(clusterIP)))
		})

		It("should remove the lighthouse config section", func() {
			Expect(strings.TrimSpace(t.assertCoreDNSConfigMap().Data["Corefile"])).To(Equal(coreDNSCorefileData("")))
		})

		t.testServiceDiscoveryDeleted()
	})

	When("the openshift DNS config exists", func() {
		BeforeEach(func() {
			t.InitClientObjs = append(t.InitClientObjs, newDNSConfig(clusterIP))
		})

		It("should remove the lighthouse config", func() {
			assertDNSConfigServers(t.assertDNSConfig(), newDNSConfig(""))
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
				t.createConfigMap(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      t.serviceDiscovery.Spec.CoreDNSCustomConfig.ConfigMapName,
						Namespace: t.serviceDiscovery.Spec.CoreDNSCustomConfig.Namespace,
					},
					Data: map[string]string{
						"lighthouse.server": strings.ReplaceAll(coreDNSConfigFormat, "$IP", clusterIP),
					},
				})
			})

			It("should remove the lighthouse config section", func() {
				Expect(t.assertConfigMap(t.serviceDiscovery.Spec.CoreDNSCustomConfig.ConfigMapName,
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
		t.serviceDiscovery.SetFinalizers([]string{constants.CleanupFinalizer})

		now := metav1.Now()
		t.serviceDiscovery.SetDeletionTimestamp(&now)
	})

	Context("", func() {
		BeforeEach(func() {
			t.InitClientObjs = append(t.InitClientObjs,
				t.NewDeployment(names.ServiceDiscoveryComponent))
		})

		It("should run a Deployment to uninstall the service discovery component", func() {
			t.AssertReconcileRequeue()

			t.AssertNoDeployment(names.ServiceDiscoveryComponent)

			t.UpdateDeploymentToReady(t.assertUninstallServiceDiscoveryDeployment())

			t.awaitFinalizer()

			t.AssertReconcileSuccess()

			t.AssertNoDeployment(names.AppendUninstall(names.ServiceDiscoveryComponent))

			t.awaitServiceDiscoveryDeleted()

			t.AssertReconcileSuccess()
			t.AssertNoDeployment(names.AppendUninstall(names.ServiceDiscoveryComponent))
		})
	})

	When("the version of the deleting ServiceDiscovery instance does not support uninstall", func() {
		BeforeEach(func() {
			t.serviceDiscovery.Spec.Version = "0.11.1"

			t.InitClientObjs = append(t.InitClientObjs, t.NewDeployment(names.ServiceDiscoveryComponent))
		})

		It("should not perform uninstall", func() {
			t.AssertReconcileSuccess()

			_, err := t.GetDeployment(names.ServiceDiscoveryComponent)
			Expect(err).To(Succeed())

			t.AssertNoDeployment(names.AppendUninstall(names.ServiceDiscoveryComponent))

			t.awaitServiceDiscoveryDeleted()
		})
	})
}
