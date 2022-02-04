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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/test"
	submariner_v1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/constants"
	"github.com/submariner-io/submariner-operator/controllers/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Service discovery controller", func() {
	Context("Reconciliation", testReconciliation)
	Context("Deletion", testDeletion)
})

func testReconciliation() {
	t := newTestDriver()

	When("the openshift DNS config exists", func() {
		Context("and the lighthouse config isn't present", func() {
			BeforeEach(func() {
				t.initClientObjs = append(t.initClientObjs, newDNSConfig(""), newDNSService(clusterIP))
			})

			It("should add it", func() {
				t.assertReconcileSuccess()

				assertDNSConfigServers(t.assertDNSConfig(), newDNSConfig(clusterIP))
			})
		})

		Context("and the lighthouse config is present and the lighthouse DNS service IP is updated", func() {
			updatedClusterIP := "10.10.10.11"

			BeforeEach(func() {
				t.initClientObjs = append(t.initClientObjs, newDNSConfig(clusterIP), newDNSService(updatedClusterIP))
			})

			It("should update the lighthouse config", func() {
				t.assertReconcileSuccess()

				assertDNSConfigServers(t.assertDNSConfig(), newDNSConfig(updatedClusterIP))
			})
		})

		Context("and the lighthouse DNS service doesn't exist", func() {
			BeforeEach(func() {
				t.initClientObjs = append(t.initClientObjs, newDNSConfig(""))
			})

			It("should create the service and add the lighthouse config", func() {
				t.assertReconcileError()

				t.setLighthouseCoreDNSServiceIP()

				t.assertReconcileSuccess()

				assertDNSConfigServers(t.assertDNSConfig(), newDNSConfig(clusterIP))
			})
		})
	})

	When("the coredns ConfigMap exists", func() {
		Context("and the lighthouse config isn't present", func() {
			BeforeEach(func() {
				t.initClientObjs = append(t.initClientObjs, newDNSService(clusterIP))
				t.createConfigMap(newCoreDNSConfigMap(coreDNSCorefileData("")))
			})

			It("should add it", func() {
				t.assertReconcileSuccess()

				Expect(strings.TrimSpace(t.assertCoreDNSConfigMap().Data["Corefile"])).To(Equal(coreDNSCorefileData(clusterIP)))
			})
		})

		Context("and the lighthouse config is present and the lighthouse DNS service IP is updated", func() {
			updatedClusterIP := "10.10.10.11"

			BeforeEach(func() {
				t.initClientObjs = append(t.initClientObjs, newDNSService(updatedClusterIP))
				t.createConfigMap(newCoreDNSConfigMap(coreDNSCorefileData(clusterIP)))
			})

			It("should update the lighthouse config", func() {
				t.assertReconcileSuccess()

				Expect(strings.TrimSpace(t.assertCoreDNSConfigMap().Data["Corefile"])).To(Equal(coreDNSCorefileData(updatedClusterIP)))
			})
		})

		Context("and the lighthouse DNS service doesn't exist", func() {
			BeforeEach(func() {
				t.createConfigMap(newCoreDNSConfigMap(coreDNSCorefileData("")))
			})

			It("should create the service and add the lighthouse config", func() {
				t.assertReconcileError()

				t.setLighthouseCoreDNSServiceIP()

				t.assertReconcileSuccess()

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

				t.initClientObjs = append(t.initClientObjs, newDNSService(clusterIP))
			})

			It("should update it", func() {
				t.assertReconcileSuccess()

				Expect(strings.TrimSpace(t.assertConfigMap(t.serviceDiscovery.Spec.CoreDNSCustomConfig.ConfigMapName,
					t.serviceDiscovery.Spec.CoreDNSCustomConfig.Namespace).Data["lighthouse.server"])).To(Equal(
					strings.ReplaceAll(lighthouseDNSConfigFormat, "$IP", clusterIP)))
			})
		})

		Context("and the custom coredns ConfigMap doesn't exist", func() {
			BeforeEach(func() {
				t.initClientObjs = append(t.initClientObjs, newDNSService(clusterIP))
			})

			It("should create it", func() {
				t.assertReconcileSuccess()

				Expect(strings.TrimSpace(t.assertConfigMap(t.serviceDiscovery.Spec.CoreDNSCustomConfig.ConfigMapName,
					t.serviceDiscovery.Spec.CoreDNSCustomConfig.Namespace).Data["lighthouse.server"])).To(Equal(
					strings.ReplaceAll(lighthouseDNSConfigFormat, "$IP", clusterIP)))
			})
		})

		Context("and the lighthouse DNS service doesn't exist", func() {
			It("should create the service and the custom coredns ConfigMap", func() {
				t.assertReconcileError()

				t.setLighthouseCoreDNSServiceIP()

				t.assertReconcileSuccess()

				Expect(strings.TrimSpace(t.assertConfigMap(t.serviceDiscovery.Spec.CoreDNSCustomConfig.ConfigMapName,
					t.serviceDiscovery.Spec.CoreDNSCustomConfig.Namespace).Data["lighthouse.server"])).To(Equal(
					strings.ReplaceAll(lighthouseDNSConfigFormat, "$IP", clusterIP)))
			})
		})
	})
}

func testDeletion() {
	t := newTestDriver()

	BeforeEach(func() {
		t.serviceDiscovery.SetFinalizers([]string{constants.CleanupFinalizer})

		now := metav1.Now()
		t.serviceDiscovery.SetDeletionTimestamp(&now)
	})

	JustBeforeEach(func() {
		t.assertReconcileSuccess()
	})

	When("the coredns ConfigMap exists", func() {
		BeforeEach(func() {
			t.createConfigMap(newCoreDNSConfigMap(coreDNSCorefileData(clusterIP)))
		})

		It("should remove the lighthouse config section", func() {
			Expect(strings.TrimSpace(t.assertCoreDNSConfigMap().Data["Corefile"])).To(Equal(coreDNSCorefileData("")))

			test.AwaitNoFinalizer(resource.ForControllerClient(t.fakeClient, submarinerNamespace, &submariner_v1.ServiceDiscovery{}),
				serviceDiscoveryName, constants.CleanupFinalizer)
		})

		t.testFinalizerRemoved()
	})

	When("the openshift DNS config exists", func() {
		BeforeEach(func() {
			t.initClientObjs = append(t.initClientObjs, newDNSConfig(clusterIP))
		})

		It("should remove the lighthouse config", func() {
			assertDNSConfigServers(t.assertDNSConfig(), newDNSConfig(""))
		})

		t.testFinalizerRemoved()
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

			t.testFinalizerRemoved()
		})

		Context("and the custom coredns ConfigMap doesn't exist", func() {
			t.testFinalizerRemoved()
		})
	})
}
