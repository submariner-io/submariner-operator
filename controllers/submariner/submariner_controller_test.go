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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	operatorv1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	submarinerName          = "submariner"
	submarinerNamespace     = "submariner-operator"
	gatewayDaemonSetName    = "submariner-gateway"
	routeAgentDaemonSetName = "submariner-routeagent"
)

var _ = Describe("Submariner controller tests", func() {
	Context("Reconciliation", testReconciliation)
})

const (
	testDetectedServiceCIDR   = "100.94.0.0/16"
	testDetectedClusterCIDR   = "10.244.0.0/16"
	testConfiguredServiceCIDR = "192.168.66.0/24"
	testConfiguredClusterCIDR = "192.168.67.0/24"
)

func testReconciliation() {
	t := newTestDriver()

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
			t.assertGatewayDaemonSet(t.withNetworkDiscovery())
		})
	})

	When("the submariner gateway DaemonSet already exists", func() {
		BeforeEach(func() {
			t.initClientObjs = append(t.initClientObjs, &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: t.submariner.Namespace,
					Name:      gatewayDaemonSetName,
				},
			})
		})

		It("should update it", func() {
			t.assertReconcileSuccess()

			initial := t.getSubmariner()
			initial.Spec.ServiceCIDR = "101.96.1.0/16"
			Expect(t.fakeClient.Update(context.TODO(), initial)).To(Succeed())

			t.assertReconcileSuccess()

			updatedDaemonSet := t.assertDaemonSet(gatewayDaemonSetName)
			Expect(envMapFrom(updatedDaemonSet)).To(HaveKeyWithValue("SUBMARINER_SERVICECIDR", initial.Spec.ServiceCIDR))
		})
	})

	When("the submariner route-agent DaemonSet doesn't exist", func() {
		It("should create it", func() {
			t.assertReconcileSuccess()
			t.assertRouteAgentDaemonSet(t.withNetworkDiscovery())
		})
	})

	When("the submariner route-agent DaemonSet already exists", func() {
		BeforeEach(func() {
			t.initClientObjs = append(t.initClientObjs, &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: t.submariner.Namespace,
					Name:      routeAgentDaemonSetName,
				},
			})
		})

		It("should update it", func() {
			t.assertReconcileSuccess()

			initial := t.getSubmariner()
			initial.Spec.ClusterCIDR = "11.245.1.0/16"
			Expect(t.fakeClient.Update(context.TODO(), initial)).To(Succeed())

			t.assertReconcileSuccess()

			updatedDaemonSet := t.assertDaemonSet(routeAgentDaemonSetName)
			Expect(envMapFrom(updatedDaemonSet)).To(HaveKeyWithValue("SUBMARINER_CLUSTERCIDR", initial.Spec.ClusterCIDR))
		})

		When("a selected pod has a nil Started field", func() {
			BeforeEach(func() {
				t.initClientObjs = append(t.initClientObjs, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: t.submariner.Namespace,
						Name:      routeAgentDaemonSetName + "-pod",
						Labels:    map[string]string{"app": "submariner-routeagent"},
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

	When("the Submariner resource doesn't exist", func() {
		BeforeEach(func() {
			t.initClientObjs = nil
		})

		It("should return success without creating any resources", func() {
			t.assertReconcileSuccess()
			t.assertNoDaemonSet(gatewayDaemonSetName)
			t.assertNoDaemonSet(routeAgentDaemonSetName)
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
