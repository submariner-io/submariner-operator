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
	"context"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	operatorv1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/constants"
	"github.com/submariner-io/submariner-operator/controllers/test"
	"github.com/submariner-io/submariner-operator/controllers/uninstall"
	"github.com/submariner-io/submariner-operator/pkg/names"
	routeagent "github.com/submariner-io/submariner/pkg/routeagent_driver/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	submarinerName      = "test-submariner"
	submarinerNamespace = "test-ns"
)

var _ = Describe("Submariner controller tests", func() {
	Context("Reconciliation", testReconciliation)
	When("the Submariner resource is being deleted", testDeletion)
})

const (
	testDetectedServiceCIDR   = "100.94.0.0/16"
	testDetectedClusterCIDR   = "10.244.0.0/16"
	testConfiguredServiceCIDR = "192.168.66.0/24"
	testConfiguredClusterCIDR = "192.168.67.0/24"
)

func testReconciliation() {
	t := newTestDriver()

	It("should add a finalizer to the Submariner resource", func() {
		t.AssertReconcileSuccess()
		t.awaitFinalizer()
	})

	When("the network details are not provided", func() {
		It("should use the detected network", func() {
			t.AssertReconcileSuccess()

			updated := t.getSubmariner()
			Expect(updated.Status.ServiceCIDR).To(Equal(testDetectedServiceCIDR))
			Expect(updated.Status.ClusterCIDR).To(Equal(testDetectedClusterCIDR))
		})
	})

	When("the network details are provided", func() {
		It("should use the provided ones instead of the detected ones", func() {
			t.AssertReconcileSuccess()

			initial := t.getSubmariner()
			initial.Spec.ServiceCIDR = testConfiguredServiceCIDR
			initial.Spec.ClusterCIDR = testConfiguredClusterCIDR

			Expect(t.Client.Update(context.TODO(), initial)).To(Succeed())

			t.AssertReconcileSuccess()

			updated := t.getSubmariner()
			Expect(updated.Status.ServiceCIDR).To(Equal(testConfiguredServiceCIDR))
			Expect(updated.Status.ClusterCIDR).To(Equal(testConfiguredClusterCIDR))
		})
	})

	When("the submariner gateway DaemonSet doesn't exist", func() {
		It("should create it", func() {
			t.AssertReconcileSuccess()
			t.assertGatewayDaemonSet()
		})
	})

	When("the submariner gateway DaemonSet already exists", func() {
		BeforeEach(func() {
			t.InitClientObjs = append(t.InitClientObjs, t.NewDaemonSet(names.GatewayComponent))
		})

		It("should update it", func() {
			t.AssertReconcileSuccess()

			initial := t.getSubmariner()
			initial.Spec.ServiceCIDR = "101.96.1.0/16"
			Expect(t.Client.Update(context.TODO(), initial)).To(Succeed())

			t.AssertReconcileSuccess()

			updatedDaemonSet := t.AssertDaemonSet(names.GatewayComponent)
			Expect(test.EnvMapFrom(updatedDaemonSet)).To(HaveKeyWithValue("SUBMARINER_SERVICECIDR", initial.Spec.ServiceCIDR))
		})
	})

	When("the submariner route-agent DaemonSet doesn't exist", func() {
		It("should create it", func() {
			t.AssertReconcileSuccess()
			t.assertRouteAgentDaemonSet()
		})
	})

	When("the submariner route-agent DaemonSet already exists", func() {
		BeforeEach(func() {
			t.InitClientObjs = append(t.InitClientObjs, t.NewDaemonSet(names.RouteAgentComponent))
		})

		It("should update it", func() {
			t.AssertReconcileSuccess()

			initial := t.getSubmariner()
			initial.Spec.ClusterCIDR = "11.245.1.0/16"
			Expect(t.Client.Update(context.TODO(), initial)).To(Succeed())

			t.AssertReconcileSuccess()

			updatedDaemonSet := t.AssertDaemonSet(names.RouteAgentComponent)
			Expect(test.EnvMapFrom(updatedDaemonSet)).To(HaveKeyWithValue("SUBMARINER_CLUSTERCIDR", initial.Spec.ClusterCIDR))
		})

		When("a selected pod has a nil Started field", func() {
			BeforeEach(func() {
				t.InitClientObjs = append(t.InitClientObjs, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: t.submariner.Namespace,
						Name:      names.RouteAgentComponent + "-pod",
						Labels:    map[string]string{"app": names.RouteAgentComponent},
					},
					Spec: corev1.PodSpec{},
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{
							{
								State: corev1.ContainerState{
									Waiting: &corev1.ContainerStateWaiting{},
								},
							},
						},
					},
				})
			})

			It("should not crash", func() {
				t.AssertReconcileSuccess()
			})
		})
	})

	When("the submariner globalnet DaemonSet doesn't exist", func() {
		It("should create it", func() {
			t.AssertReconcileSuccess()
			t.assertGlobalnetDaemonSet()
		})
	})

	When("the submariner network plugin syncer Deployment doesn't exist", func() {
		BeforeEach(func() {
			t.clusterNetwork.NetworkPlugin = routeagent.NetworkPluginOVNKubernetes
		})

		It("should create it", func() {
			t.AssertReconcileSuccess()
			t.assertNetworkPluginSyncerDeployment()
		})
	})

	When("the Submariner resource doesn't exist", func() {
		BeforeEach(func() {
			t.InitClientObjs = nil
		})

		It("should return success without creating any resources", func() {
			t.AssertReconcileSuccess()
			t.AssertNoDaemonSet(names.GatewayComponent)
			t.AssertNoDaemonSet(names.RouteAgentComponent)
		})
	})

	When("the Submariner resource is missing values for certain fields", func() {
		BeforeEach(func() {
			t.submariner.Spec.Repository = ""
			t.submariner.Spec.Version = ""
		})

		It("should update the resource with defaults", func() {
			t.AssertReconcileSuccess()

			updated := t.getSubmariner()
			Expect(updated.Spec.Repository).To(Equal(operatorv1.DefaultRepo))
			Expect(updated.Spec.Version).To(Equal(operatorv1.DefaultSubmarinerVersion))
		})
	})

	When("DaemonSet creation fails", func() {
		BeforeEach(func() {
			t.Client = &test.FailingClient{Client: t.NewClient(), OnCreate: reflect.TypeOf(&appsv1.DaemonSet{})}
		})

		It("should return an error", func() {
			_, err := t.DoReconcile()
			Expect(err).To(HaveOccurred())
		})
	})

	When("DaemonSet retrieval fails", func() {
		BeforeEach(func() {
			t.Client = &test.FailingClient{Client: t.NewClient(), OnGet: reflect.TypeOf(&appsv1.DaemonSet{})}
		})

		It("should return an error", func() {
			_, err := t.DoReconcile()
			Expect(err).To(HaveOccurred())
		})
	})

	When("Submariner resource retrieval fails", func() {
		BeforeEach(func() {
			t.Client = &test.FailingClient{Client: t.NewClient(), OnGet: reflect.TypeOf(&operatorv1.Submariner{})}
		})

		It("should return an error", func() {
			_, err := t.DoReconcile()
			Expect(err).To(HaveOccurred())
		})
	})
}

func testDeletion() {
	t := newTestDriver()

	BeforeEach(func() {
		t.submariner.SetFinalizers([]string{constants.CleanupFinalizer})

		now := metav1.Now()
		t.submariner.SetDeletionTimestamp(&now)
	})

	Context("", func() {
		BeforeEach(func() {
			t.clusterNetwork.NetworkPlugin = routeagent.NetworkPluginOVNKubernetes

			t.InitClientObjs = append(t.InitClientObjs,
				t.NewDaemonSet(names.GatewayComponent),
				t.NewPodWithLabel("app", names.GatewayComponent),
				t.NewDaemonSet(names.RouteAgentComponent),
				t.NewDaemonSet(names.GlobalnetComponent),
				t.NewDeployment(names.NetworkPluginSyncerComponent))
		})

		It("should run DaemonSets/Deployments to uninstall components", func() {
			// The first reconcile invocation should delete the regular DaemonSets/Deployments.
			t.AssertReconcileRequeue()

			t.AssertNoDaemonSet(names.GatewayComponent)
			t.AssertNoDaemonSet(names.RouteAgentComponent)
			t.AssertNoDaemonSet(names.GlobalnetComponent)
			t.AssertNoDeployment(names.NetworkPluginSyncerComponent)

			// Simulate the gateway DaemonSet controller cleaning up its pods.
			t.DeletePods("app", names.GatewayComponent)

			// Next, the controller should create the corresponding uninstall DaemonSets/Deployments.
			t.AssertReconcileRequeue()

			// For the globalnet DaemonSet, we'll only update it to nodes available but not yet ready at this point.
			globalnetDS := t.assertUninstallGlobalnetDaemonSet()
			t.UpdateDaemonSetToScheduled(globalnetDS)

			// For the gateway DaemonSet, we'll update it to observed but no nodes available - this will cause it to be deleted.
			t.UpdateDaemonSetToObserved(t.assertUninstallGatewayDaemonSet())

			t.UpdateDaemonSetToReady(t.assertUninstallRouteAgentDaemonSet())
			t.UpdateDeploymentToReady(t.assertUninstallNetworkPluginSyncerDeployment())

			// Next, the controller should again requeue b/c the gateway DaemonSet isn't ready yet.
			t.AssertReconcileRequeue()

			// Now update the globalnet DaemonSet to ready.
			t.UpdateDaemonSetToReady(globalnetDS)

			// Ensure the finalizer is still present.
			t.awaitFinalizer()

			// Finally, the controller should delete the uninstall DaemonSets/Deployments and remove the finalizer.
			t.AssertReconcileSuccess()

			t.AssertNoDaemonSet(names.AppendUninstall(names.GatewayComponent))
			t.AssertNoDaemonSet(names.AppendUninstall(names.RouteAgentComponent))
			t.AssertNoDaemonSet(names.AppendUninstall(names.GlobalnetComponent))
			t.AssertNoDeployment(names.AppendUninstall(names.NetworkPluginSyncerComponent))

			t.awaitSubmarinerDeleted()

			t.AssertReconcileSuccess()
			t.AssertNoDaemonSet(names.AppendUninstall(names.GatewayComponent))
		})
	})

	Context("and some components aren't installed", func() {
		BeforeEach(func() {
			t.submariner.Spec.GlobalCIDR = ""
			t.submariner.Spec.Version = "devel"

			t.InitClientObjs = append(t.InitClientObjs,
				t.NewDaemonSet(names.GatewayComponent),
				t.NewDaemonSet(names.RouteAgentComponent))
		})

		It("should only create uninstall DaemonSets/Deployments for installed components", func() {
			t.AssertReconcileRequeue()

			t.UpdateDaemonSetToReady(t.assertUninstallGatewayDaemonSet())
			t.UpdateDaemonSetToReady(t.assertUninstallRouteAgentDaemonSet())

			t.AssertNoDaemonSet(names.AppendUninstall(names.GlobalnetComponent))
			t.AssertNoDaemonSet(names.AppendUninstall(names.NetworkPluginSyncerComponent))

			t.AssertReconcileSuccess()

			t.AssertNoDaemonSet(names.AppendUninstall(names.GatewayComponent))
			t.AssertNoDaemonSet(names.AppendUninstall(names.RouteAgentComponent))

			t.awaitSubmarinerDeleted()
		})
	})

	Context("and an uninstall DaemonSet does not complete in time", func() {
		BeforeEach(func() {
			t.submariner.Spec.GlobalCIDR = ""
		})

		It("should delete it", func() {
			t.AssertReconcileRequeue()

			t.UpdateDaemonSetToReady(t.assertUninstallGatewayDaemonSet())
			t.UpdateDaemonSetToScheduled(t.assertUninstallRouteAgentDaemonSet())

			ts := metav1.NewTime(time.Now().Add(-(uninstall.ComponentReadyTimeout + 10)))
			t.submariner.SetDeletionTimestamp(&ts)
			Expect(t.Client.Update(context.TODO(), t.submariner)).To(Succeed())

			t.AssertReconcileSuccess()

			t.AssertNoDaemonSet(names.AppendUninstall(names.GatewayComponent))
			t.AssertNoDaemonSet(names.AppendUninstall(names.RouteAgentComponent))

			t.awaitSubmarinerDeleted()
		})
	})

	Context("and the version of the deleting Submariner instance does not support uninstall", func() {
		BeforeEach(func() {
			t.submariner.Spec.Version = "0.11.1"

			t.InitClientObjs = append(t.InitClientObjs,
				t.NewDaemonSet(names.GatewayComponent))
		})

		It("should not perform uninstall", func() {
			t.AssertReconcileSuccess()

			_, err := t.GetDaemonSet(names.GatewayComponent)
			Expect(err).To(Succeed())

			t.AssertNoDaemonSet(names.AppendUninstall(names.GatewayComponent))

			t.awaitSubmarinerDeleted()
		})
	})

	Context("and ServiceDiscovery is enabled", func() {
		BeforeEach(func() {
			t.submariner.Spec.GlobalCIDR = ""
			t.submariner.Spec.ServiceDiscoveryEnabled = true

			t.InitClientObjs = append(t.InitClientObjs,
				&operatorv1.ServiceDiscovery{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: submarinerNamespace,
						Name:      names.ServiceDiscoveryCrName,
					},
				})
		})

		It("should delete the ServiceDiscovery resource", func() {
			t.AssertReconcileRequeue()

			t.UpdateDaemonSetToReady(t.assertUninstallGatewayDaemonSet())
			t.UpdateDaemonSetToReady(t.assertUninstallRouteAgentDaemonSet())

			t.AssertReconcileSuccess()

			serviceDiscovery := &operatorv1.ServiceDiscovery{}
			err := t.Client.Get(context.TODO(), types.NamespacedName{Name: names.ServiceDiscoveryCrName, Namespace: submarinerNamespace},
				serviceDiscovery)
			Expect(errors.IsNotFound(err)).To(BeTrue(), "ServiceDiscovery still exists")

			t.awaitSubmarinerDeleted()
		})
	})
}
