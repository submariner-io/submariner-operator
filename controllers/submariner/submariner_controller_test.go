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
	"github.com/submariner-io/admiral/pkg/test"
	operatorv1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/constants"
	"github.com/submariner-io/submariner-operator/controllers/resource"
	"github.com/submariner-io/submariner-operator/controllers/uninstall"
	"github.com/submariner-io/submariner-operator/pkg/names"
	routeagent "github.com/submariner-io/submariner/pkg/routeagent_driver/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		t.assertReconcileSuccess()
		test.AwaitFinalizer(resource.ForControllerClient(t.fakeClient, submarinerNamespace, &operatorv1.Submariner{}),
			submarinerName, constants.CleanupFinalizer)
	})

	When("the network details are not provided", func() {
		It("should use the detected network", func() {
			t.assertReconcileSuccess()

			updated := t.getSubmariner()
			Expect(updated.Status.ServiceCIDR).To(Equal(testDetectedServiceCIDR))
			Expect(updated.Status.ClusterCIDR).To(Equal(testDetectedClusterCIDR))
		})
	})

	When("the network details are provided", func() {
		It("should use the provided ones instead of the detected ones", func() {
			t.assertReconcileSuccess()

			initial := t.getSubmariner()
			initial.Spec.ServiceCIDR = testConfiguredServiceCIDR
			initial.Spec.ClusterCIDR = testConfiguredClusterCIDR

			Expect(t.fakeClient.Update(context.TODO(), initial)).To(Succeed())

			t.assertReconcileSuccess()

			updated := t.getSubmariner()
			Expect(updated.Status.ServiceCIDR).To(Equal(testConfiguredServiceCIDR))
			Expect(updated.Status.ClusterCIDR).To(Equal(testConfiguredClusterCIDR))
		})
	})

	When("the submariner gateway DaemonSet doesn't exist", func() {
		It("should create it", func() {
			t.assertReconcileSuccess()
			t.assertGatewayDaemonSet()
		})
	})

	When("the submariner gateway DaemonSet already exists", func() {
		BeforeEach(func() {
			t.initClientObjs = append(t.initClientObjs, t.newDaemonSet(names.GatewayComponent))
		})

		It("should update it", func() {
			t.assertReconcileSuccess()

			initial := t.getSubmariner()
			initial.Spec.ServiceCIDR = "101.96.1.0/16"
			Expect(t.fakeClient.Update(context.TODO(), initial)).To(Succeed())

			t.assertReconcileSuccess()

			updatedDaemonSet := t.assertDaemonSet(names.GatewayComponent)
			Expect(envMapFrom(updatedDaemonSet)).To(HaveKeyWithValue("SUBMARINER_SERVICECIDR", initial.Spec.ServiceCIDR))
		})
	})

	When("the submariner route-agent DaemonSet doesn't exist", func() {
		It("should create it", func() {
			t.assertReconcileSuccess()
			t.assertRouteAgentDaemonSet()
		})
	})

	When("the submariner route-agent DaemonSet already exists", func() {
		BeforeEach(func() {
			t.initClientObjs = append(t.initClientObjs, t.newDaemonSet(names.RouteAgentComponent))
		})

		It("should update it", func() {
			t.assertReconcileSuccess()

			initial := t.getSubmariner()
			initial.Spec.ClusterCIDR = "11.245.1.0/16"
			Expect(t.fakeClient.Update(context.TODO(), initial)).To(Succeed())

			t.assertReconcileSuccess()

			updatedDaemonSet := t.assertDaemonSet(names.RouteAgentComponent)
			Expect(envMapFrom(updatedDaemonSet)).To(HaveKeyWithValue("SUBMARINER_CLUSTERCIDR", initial.Spec.ClusterCIDR))
		})

		When("a selected pod has a nil Started field", func() {
			BeforeEach(func() {
				t.initClientObjs = append(t.initClientObjs, &corev1.Pod{
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
				t.assertReconcileSuccess()
			})
		})
	})

	When("the submariner globalnet DaemonSet doesn't exist", func() {
		It("should create it", func() {
			t.assertReconcileSuccess()
			t.assertGlobalnetDaemonSet()
		})
	})

	When("the submariner network plugin syncer Deployment doesn't exist", func() {
		BeforeEach(func() {
			t.clusterNetwork.NetworkPlugin = routeagent.NetworkPluginOVNKubernetes
		})

		It("should create it", func() {
			t.assertReconcileSuccess()
			t.assertNetworkPluginSyncerDeployment()
		})
	})

	When("the Submariner resource doesn't exist", func() {
		BeforeEach(func() {
			t.initClientObjs = nil
		})

		It("should return success without creating any resources", func() {
			t.assertReconcileSuccess()
			t.assertNoDaemonSet(names.GatewayComponent)
			t.assertNoDaemonSet(names.RouteAgentComponent)
		})
	})

	When("the Submariner resource is missing values for certain fields", func() {
		BeforeEach(func() {
			t.submariner.Spec.Repository = ""
			t.submariner.Spec.Version = ""
		})

		It("should update the resource with defaults", func() {
			t.assertReconcileSuccess()

			updated := t.getSubmariner()
			Expect(updated.Spec.Repository).To(Equal(operatorv1.DefaultRepo))
			Expect(updated.Spec.Version).To(Equal(operatorv1.DefaultSubmarinerVersion))
		})
	})

	When("DaemonSet creation fails", func() {
		BeforeEach(func() {
			t.fakeClient = &failingClient{Client: t.newClient(), onCreate: reflect.TypeOf(&appsv1.DaemonSet{})}
		})

		It("should return an error", func() {
			_, err := t.doReconcile()
			Expect(err).To(HaveOccurred())
		})
	})

	When("DaemonSet retrieval fails", func() {
		BeforeEach(func() {
			t.fakeClient = &failingClient{Client: t.newClient(), onGet: reflect.TypeOf(&appsv1.DaemonSet{})}
		})

		It("should return an error", func() {
			_, err := t.doReconcile()
			Expect(err).To(HaveOccurred())
		})
	})

	When("Submariner resource retrieval fails", func() {
		BeforeEach(func() {
			t.fakeClient = &failingClient{Client: t.newClient(), onGet: reflect.TypeOf(&operatorv1.Submariner{})}
		})

		It("should return an error", func() {
			_, err := t.doReconcile()
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

			t.initClientObjs = append(t.initClientObjs,
				t.newDaemonSet(names.GatewayComponent),
				t.newPodWithLabel("app", names.GatewayComponent),
				t.newDaemonSet(names.RouteAgentComponent),
				t.newDaemonSet(names.GlobalnetComponent),
				t.newDeployment(names.NetworkPluginSyncerComponent))
		})

		It("should run DaemonSets/Deployments to uninstall components", func() {
			// The first reconcile invocation should delete the regular DaemonSets/Deployments.
			t.assertReconcileRequeue()

			t.assertNoDaemonSet(names.GatewayComponent)
			t.assertNoDaemonSet(names.RouteAgentComponent)
			t.assertNoDaemonSet(names.GlobalnetComponent)
			t.assertNoDeployment(names.NetworkPluginSyncerComponent)

			// Simulate the DaemonSet controller cleaning up its pods.
			t.deletePods("app", names.GatewayComponent)

			// Next, the controller should create the corresponding uninstall DaemonSets/Deployments.
			t.assertReconcileRequeue()

			// For the gateway DaemonSet, we'll only update it to nodes available but not yet ready at this point.
			gatewayDS := t.assertUninstallGatewayDaemonSet()
			t.updateDaemonSetToScheduled(gatewayDS)

			// For the globalnet DaemonSet, we'll update it to observed but no nodes available - this will cause it to be deleted.
			globalnetDS := t.assertUninstallGlobalnetDaemonSet()
			t.updateDaemonSetToObserved(globalnetDS)

			t.updateDaemonSetToReady(t.assertUninstallRouteAgentDaemonSet())
			t.updateDeploymentToReady(t.assertUninstallNetworkPluginSyncerDeployment())

			// Next, the controller should again requeue b/c the gateway DaemonSet isn't ready yet.
			t.assertReconcileRequeue()

			// Now update the gateway DaemonSet to ready.
			t.updateDaemonSetToReady(gatewayDS)

			// Ensure the finalizer is still present.
			test.AwaitFinalizer(resource.ForControllerClient(t.fakeClient, submarinerNamespace, &operatorv1.Submariner{}),
				submarinerName, constants.CleanupFinalizer)

			// Finally, the controller should delete the uninstall DaemonSets/Deployments and remove the finalizer.
			t.assertReconcileSuccess()

			t.assertNoDaemonSet(names.AppendUninstall(names.GatewayComponent))
			t.assertNoDaemonSet(names.AppendUninstall(names.RouteAgentComponent))
			t.assertNoDaemonSet(names.AppendUninstall(names.GlobalnetComponent))
			t.assertNoDeployment(names.AppendUninstall(names.NetworkPluginSyncerComponent))

			test.AwaitNoFinalizer(resource.ForControllerClient(t.fakeClient, submarinerNamespace, &operatorv1.Submariner{}),
				submarinerName, constants.CleanupFinalizer)
		})
	})

	Context("and some components aren't installed", func() {
		BeforeEach(func() {
			t.submariner.Spec.GlobalCIDR = ""

			t.initClientObjs = append(t.initClientObjs,
				t.newDaemonSet(names.GatewayComponent),
				t.newDaemonSet(names.RouteAgentComponent))
		})

		It("should only create uninstall DaemonSets/Deployments for installed components", func() {
			t.assertReconcileRequeue()

			t.updateDaemonSetToReady(t.assertUninstallGatewayDaemonSet())
			t.updateDaemonSetToReady(t.assertUninstallRouteAgentDaemonSet())

			t.assertNoDaemonSet(names.AppendUninstall(names.GlobalnetComponent))
			t.assertNoDaemonSet(names.AppendUninstall(names.NetworkPluginSyncerComponent))

			t.assertReconcileSuccess()

			t.assertNoDaemonSet(names.AppendUninstall(names.GatewayComponent))
			t.assertNoDaemonSet(names.AppendUninstall(names.RouteAgentComponent))

			test.AwaitNoFinalizer(resource.ForControllerClient(t.fakeClient, submarinerNamespace, &operatorv1.Submariner{}),
				submarinerName, constants.CleanupFinalizer)
		})
	})

	Context("and an uninstall DaemonSet does not complete in time", func() {
		BeforeEach(func() {
			t.submariner.Spec.GlobalCIDR = ""
		})

		It("should delete it", func() {
			t.assertReconcileRequeue()

			t.updateDaemonSetToReady(t.assertUninstallGatewayDaemonSet())
			t.updateDaemonSetToScheduled(t.assertUninstallRouteAgentDaemonSet())

			ts := metav1.NewTime(time.Now().Add(-(uninstall.ComponentReadyTimeout + 10)))
			t.submariner.SetDeletionTimestamp(&ts)
			Expect(t.fakeClient.Update(context.TODO(), t.submariner)).To(Succeed())

			t.assertReconcileSuccess()

			t.assertNoDaemonSet(names.AppendUninstall(names.GatewayComponent))
			t.assertNoDaemonSet(names.AppendUninstall(names.RouteAgentComponent))

			test.AwaitNoFinalizer(resource.ForControllerClient(t.fakeClient, submarinerNamespace, &operatorv1.Submariner{}),
				submarinerName, constants.CleanupFinalizer)
		})
	})

	Context("and the version of the deleting Submariner instance does not support uninstall", func() {
		BeforeEach(func() {
			t.submariner.Spec.Version = "0.11.1"

			t.initClientObjs = append(t.initClientObjs,
				t.newDaemonSet(names.GatewayComponent))
		})

		It("should not perform uninstall", func() {
			t.assertReconcileSuccess()

			_, err := t.getDaemonSet(names.GatewayComponent)
			Expect(err).To(Succeed())

			t.assertNoDaemonSet(names.AppendUninstall(names.GatewayComponent))

			test.AwaitNoFinalizer(resource.ForControllerClient(t.fakeClient, submarinerNamespace, &operatorv1.Submariner{}),
				submarinerName, constants.CleanupFinalizer)
		})
	})
}
