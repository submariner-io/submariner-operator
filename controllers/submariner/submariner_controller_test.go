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
	"os"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1config "github.com/openshift/api/config/v1"
	"github.com/submariner-io/admiral/pkg/fake"
	"github.com/submariner-io/admiral/pkg/names"
	"github.com/submariner-io/admiral/pkg/resource"
	syncertest "github.com/submariner-io/admiral/pkg/syncer/test"
	testutil "github.com/submariner-io/admiral/pkg/test"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/test"
	"github.com/submariner-io/submariner-operator/controllers/uninstall"
	opnames "github.com/submariner-io/submariner-operator/pkg/names"
	"github.com/submariner-io/submariner/pkg/cni"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	It("should add a finalizer to the Submariner resource", func(ctx SpecContext) {
		t.AssertReconcileSuccess(ctx)
		t.awaitFinalizer()
	})

	Context("", func() {
		BeforeEach(func() {
			t.submariner.Spec.NatEnabled = true
			t.submariner.Spec.AirGappedDeployment = true
		})

		It("should populate general Submariner resource Status fields from the Spec", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)

			updated := t.getSubmariner(ctx)
			Expect(updated.Status.NatEnabled).To(BeTrue())
			Expect(updated.Status.AirGappedDeployment).To(BeTrue())
			Expect(updated.Status.ClusterID).To(Equal(t.submariner.Spec.ClusterID))
			Expect(updated.Status.GlobalCIDR).To(Equal(t.submariner.Spec.GlobalCIDR))
			Expect(updated.Status.NetworkPlugin).To(Equal(t.clusterNetwork.NetworkPlugin))
			Expect(updated.Status.Version).To(Equal(t.submariner.Spec.Version))
		})
	})

	When("the network details are not provided", func() {
		It("should use the detected network", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)

			updated := t.getSubmariner(ctx)
			Expect(updated.Status.ServiceCIDR).To(Equal(testDetectedServiceCIDR))
			Expect(updated.Status.ClusterCIDR).To(Equal(testDetectedClusterCIDR))
		})
	})

	When("the network details are provided", func() {
		It("should use the provided ones instead of the detected ones", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)

			initial := t.getSubmariner(ctx)
			initial.Spec.ServiceCIDR = testConfiguredServiceCIDR
			initial.Spec.ClusterCIDR = testConfiguredClusterCIDR

			Expect(t.ScopedClient.Update(ctx, initial)).To(Succeed())

			t.AssertReconcileSuccess(ctx)

			updated := t.getSubmariner(ctx)
			Expect(updated.Status.ServiceCIDR).To(Equal(testConfiguredServiceCIDR))
			Expect(updated.Status.ClusterCIDR).To(Equal(testConfiguredClusterCIDR))
		})
	})

	When("the submariner gateway DaemonSet doesn't exist", func() {
		It("should create it", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)
			t.assertGatewayDaemonSet(ctx)
		})
	})

	When("the submariner gateway DaemonSet already exists", func() {
		BeforeEach(func() {
			t.InitScopedClientObjs = append(t.InitScopedClientObjs, t.NewDaemonSet(names.GatewayComponent))
		})

		It("should update it", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)

			initial := t.getSubmariner(ctx)
			initial.Spec.ServiceCIDR = "101.96.1.0/16"
			Expect(t.ScopedClient.Update(ctx, initial)).To(Succeed())

			t.AssertReconcileSuccess(ctx)

			updatedDaemonSet := t.AssertDaemonSet(ctx, names.GatewayComponent)
			Expect(test.EnvMapFrom(updatedDaemonSet)).To(HaveKeyWithValue("SUBMARINER_SERVICECIDR", initial.Spec.ServiceCIDR))
		})
	})

	When("the submariner route-agent DaemonSet doesn't exist", func() {
		It("should create it", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)
			t.assertRouteAgentDaemonSet(ctx)
		})
	})

	When("the submariner route-agent DaemonSet already exists", func() {
		BeforeEach(func() {
			t.InitScopedClientObjs = append(t.InitScopedClientObjs, t.NewDaemonSet(names.RouteAgentComponent))
		})

		It("should update it", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)

			initial := t.getSubmariner(ctx)
			initial.Spec.ClusterCIDR = "11.245.1.0/16"
			Expect(t.ScopedClient.Update(ctx, initial)).To(Succeed())

			t.AssertReconcileSuccess(ctx)

			updatedDaemonSet := t.AssertDaemonSet(ctx, names.RouteAgentComponent)
			Expect(test.EnvMapFrom(updatedDaemonSet)).To(HaveKeyWithValue("SUBMARINER_CLUSTERCIDR", initial.Spec.ClusterCIDR))
		})

		When("a selected pod has a nil Started field", func() {
			BeforeEach(func() {
				t.InitScopedClientObjs = append(t.InitScopedClientObjs, &corev1.Pod{
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

			It("should not crash", func(ctx SpecContext) {
				t.AssertReconcileSuccess(ctx)
			})
		})
	})

	When("the submariner globalnet DaemonSet doesn't exist", func() {
		It("should create it", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)
			t.assertGlobalnetDaemonSet(ctx)
		})
	})

	When("ServiceDiscovery is enabled", func() {
		BeforeEach(func() {
			t.submariner.Spec.ServiceDiscoveryEnabled = true
		})

		It("should create the ServiceDiscovery resource", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)

			serviceDiscovery := &v1alpha1.ServiceDiscovery{}
			err := t.ScopedClient.Get(ctx, types.NamespacedName{Name: opnames.ServiceDiscoveryCrName, Namespace: submarinerNamespace},
				serviceDiscovery)
			Expect(err).To(Succeed())

			Expect(serviceDiscovery.Spec.Version).To(Equal(t.submariner.Spec.Version))
			Expect(serviceDiscovery.Spec.Repository).To(Equal(t.submariner.Spec.Repository))
			Expect(serviceDiscovery.Spec.BrokerK8sCA).To(Equal(t.submariner.Spec.BrokerK8sCA))
			Expect(serviceDiscovery.Spec.BrokerK8sRemoteNamespace).To(Equal(t.submariner.Spec.BrokerK8sRemoteNamespace))
			Expect(serviceDiscovery.Spec.BrokerK8sApiServerToken).To(Equal(t.submariner.Spec.BrokerK8sApiServerToken))
			Expect(serviceDiscovery.Spec.BrokerK8sApiServer).To(Equal(t.submariner.Spec.BrokerK8sApiServer))
			Expect(serviceDiscovery.Spec.BrokerK8sSecret).To(Equal(t.submariner.Spec.BrokerK8sSecret))
			Expect(serviceDiscovery.Spec.ClusterID).To(Equal(t.submariner.Spec.ClusterID))
			Expect(serviceDiscovery.Spec.Namespace).To(Equal(t.submariner.Spec.Namespace))
			Expect(serviceDiscovery.Spec.GlobalnetEnabled).To(BeTrue())
		})
	})

	When("load balancer is enabled", func() {
		BeforeEach(func() {
			t.submariner.Spec.LoadBalancerEnabled = true
		})

		It("should create the load balancer service", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)
			t.assertLoadBalancerService(ctx)
		})

		Context("and the Openshift platform type is AWS", func() {
			BeforeEach(func() {
				t.InitGeneralClientObjs = append(t.InitGeneralClientObjs, newInfrastructureCluster(v1config.AWSPlatformType))
			})

			It("should create the correct load balancer service", func(ctx SpecContext) {
				t.AssertReconcileSuccess(ctx)

				service := t.assertLoadBalancerService(ctx)
				Expect(service.Annotations).To(HaveKeyWithValue("service.beta.kubernetes.io/aws-load-balancer-type", "nlb"))
			})
		})

		Context("and the Openshift platform type is IBMCloud", func() {
			BeforeEach(func() {
				t.InitGeneralClientObjs = append(t.InitGeneralClientObjs, newInfrastructureCluster(v1config.IBMCloudPlatformType))
			})

			It("should create the correct load balancer service", func(ctx SpecContext) {
				t.AssertReconcileSuccess(ctx)

				service := t.assertLoadBalancerService(ctx)
				Expect(service.Annotations).To(HaveKeyWithValue(
					"service.kubernetes.io/ibm-load-balancer-cloud-provider-enable-features", "nlb"))
			})
		})
	})

	When("the broker secret is created/updated", func() {
		var brokerSecret *corev1.Secret

		BeforeEach(func() {
			t.submariner.Spec.BrokerK8sSecret = "submariner-broker-secret"

			t.getAuthorizedBrokerClientFor = func(_ *v1alpha1.SubmarinerSpec, brokerToken, brokerCA string, _ schema.GroupVersionResource,
			) (dynamic.Interface, error) {
				Expect(brokerToken).To(Equal(t.submariner.Spec.BrokerK8sApiServerToken))
				Expect(brokerCA).To(Equal(t.submariner.Spec.BrokerK8sCA))

				return t.dynClient, nil
			}

			saName := opnames.ForClusterSA(t.submariner.Spec.ClusterID)
			brokerSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      saName + "-token",
					Namespace: t.submariner.Spec.BrokerK8sRemoteNamespace,
					Annotations: map[string]string{
						corev1.ServiceAccountNameKey: saName,
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
				Data: map[string][]byte{"data": {1, 2, 3}},
			}

			syncertest.CreateResource(t.secrets.Namespace(brokerSecret.Namespace), brokerSecret)
		})

		It("should update the local secret", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)

			ri := resource.ForDynamic(t.secrets.Namespace(t.submariner.Spec.Namespace))

			obj := testutil.AwaitResource(ri, t.submariner.Spec.BrokerK8sSecret)
			secret := resource.MustFromUnstructured(obj, &corev1.Secret{})
			Expect(secret.Data).To(Equal(brokerSecret.Data))
			Expect(secret.Type).To(Equal(corev1.SecretTypeOpaque))

			brokerSecret.Data = map[string][]byte{"data": {40, 50, 60}}
			syncertest.UpdateResource(t.secrets.Namespace(brokerSecret.Namespace), brokerSecret)

			testutil.AwaitAndVerifyResource(ri, t.submariner.Spec.BrokerK8sSecret, func(obj *unstructured.Unstructured) bool {
				return reflect.DeepEqual(resource.MustFromUnstructured(obj, &corev1.Secret{}).Data, brokerSecret.Data)
			})
		})

		Context("and the local secret already exists", func() {
			BeforeEach(func() {
				t.submariner.Spec.BrokerK8sSecret = "submariner-broker-secret"

				localSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      t.submariner.Spec.BrokerK8sSecret,
						Namespace: t.Namespace,
					},
					Type: corev1.SecretTypeServiceAccountToken,
					Data: map[string][]byte{
						"token":  {11, 22, 33, 44},
						"ca.crt": {5, 6},
					},
				}

				syncertest.CreateResource(t.secrets.Namespace(t.Namespace), localSecret)

				t.getAuthorizedBrokerClientFor = func(_ *v1alpha1.SubmarinerSpec, brokerToken, _ string, _ schema.GroupVersionResource,
				) (dynamic.Interface, error) {
					Expect(brokerToken).To(Equal(string(localSecret.Data["token"])))
					return t.dynClient, nil
				}
			})

			It("should use the local secret's credentials to access the broker", func(ctx SpecContext) {
				t.AssertReconcileSuccess(ctx)
			})
		})
	})

	When("the Submariner resource doesn't exist", func() {
		BeforeEach(func() {
			t.InitScopedClientObjs = nil
		})

		It("should return success without creating any resources", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)
			t.AssertNoDaemonSet(ctx, names.GatewayComponent)
			t.AssertNoDaemonSet(ctx, names.RouteAgentComponent)
		})
	})

	When("the Submariner resource is missing values for certain fields", func() {
		BeforeEach(func() {
			t.submariner.Spec.Repository = ""
			t.submariner.Spec.Version = ""
		})

		It("should update the resource with defaults", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)

			updated := t.getSubmariner(ctx)
			Expect(updated.Spec.Repository).To(Equal(v1alpha1.DefaultRepo))
			Expect(updated.Spec.Version).To(Equal(v1alpha1.DefaultSubmarinerVersion))
		})
	})

	When("DaemonSet creation fails", func() {
		BeforeEach(func() {
			t.ScopedClient = fake.NewReactingClient(t.NewScopedClient()).AddReactor(fake.Create, &appsv1.DaemonSet{},
				fake.FailingReaction(nil))
		})

		It("should return an error", func(ctx SpecContext) {
			_, err := t.DoReconcile(ctx)
			Expect(err).To(HaveOccurred())
		})
	})

	When("DaemonSet retrieval fails", func() {
		BeforeEach(func() {
			t.ScopedClient = fake.NewReactingClient(t.NewScopedClient()).AddReactor(fake.Get, &appsv1.DaemonSet{},
				fake.FailingReaction(nil))
		})

		It("should return an error", func(ctx SpecContext) {
			_, err := t.DoReconcile(ctx)
			Expect(err).To(HaveOccurred())
		})
	})

	When("Submariner resource retrieval fails", func() {
		BeforeEach(func() {
			t.ScopedClient = fake.NewReactingClient(t.NewScopedClient()).AddReactor(fake.Get, &v1alpha1.Submariner{},
				fake.FailingReaction(nil))
		})

		It("should return an error", func(ctx SpecContext) {
			_, err := t.DoReconcile(ctx)
			Expect(err).To(HaveOccurred())
		})
	})

	When("proxy environment variables are set", func() {
		var httpProxy, httpsProxy, noProxy string
		var httpProxySet, httpsProxySet, noProxySet bool
		const testHTTPSProxy = "https://proxy.example.com"
		const testHTTPProxy = "http://proxy.example.com"
		const testNoProxy = "127.0.0.1"

		BeforeEach(func() {
			// We know we only write the all-caps versions
			httpProxy, httpProxySet = os.LookupEnv("HTTP_PROXY")
			httpsProxy, httpsProxySet = os.LookupEnv("HTTPS_PROXY")
			noProxy, noProxySet = os.LookupEnv("NO_PROXY")
			os.Setenv("HTTPS_PROXY", testHTTPSProxy)
			os.Setenv("HTTP_PROXY", testHTTPProxy)
			os.Setenv("NO_PROXY", testNoProxy)
		})

		AfterEach(func() {
			restoreOrUnsetEnv("HTTPS_PROXY", httpsProxySet, httpsProxy)
			restoreOrUnsetEnv("HTTP_PROXY", httpProxySet, httpProxy)
			restoreOrUnsetEnv("NO_PROXY", noProxySet, noProxy)
		})

		It("should populate them in generated container specs", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)

			for _, component := range []string{
				names.GatewayComponent, names.GlobalnetComponent, names.MetricsProxyComponent, names.RouteAgentComponent,
			} {
				daemonSet := t.AssertDaemonSet(ctx, component)
				envMap := test.EnvMapFrom(daemonSet)
				Expect(envMap).To(HaveKeyWithValue("HTTPS_PROXY", testHTTPSProxy))
				Expect(envMap).To(HaveKeyWithValue("HTTP_PROXY", testHTTPProxy))
				Expect(envMap).To(HaveKeyWithValue("NO_PROXY", testNoProxy))
			}
		})
	})
}

func restoreOrUnsetEnv(envVar string, wasSet bool, value string) {
	if wasSet {
		os.Setenv(envVar, value)
	} else {
		os.Unsetenv(envVar)
	}
}

func (t *testDriver) assertLoadBalancerService(ctx context.Context) *corev1.Service {
	service := &corev1.Service{}
	err := t.ScopedClient.Get(ctx, types.NamespacedName{Name: "submariner-gateway", Namespace: submarinerNamespace},
		service)
	Expect(err).To(Succeed())
	Expect(service.Spec.Type).To(Equal(corev1.ServiceTypeLoadBalancer))

	return service
}

func testDeletion() {
	t := newTestDriver()

	var deletionTimestamp metav1.Time

	BeforeEach(func() {
		t.submariner.SetFinalizers([]string{opnames.CleanupFinalizer})

		deletionTimestamp = metav1.Now()
		t.submariner.SetDeletionTimestamp(&deletionTimestamp)
	})

	Context("", func() {
		BeforeEach(func() {
			t.clusterNetwork.NetworkPlugin = cni.OVNKubernetes

			t.InitScopedClientObjs = append(t.InitScopedClientObjs,
				t.NewDaemonSet(names.GatewayComponent),
				t.NewPodWithLabel("app", names.GatewayComponent),
				t.NewDaemonSet(names.RouteAgentComponent),
				t.NewDaemonSet(names.GlobalnetComponent))
		})

		It("should run DaemonSets/Deployments to uninstall components", func(ctx SpecContext) {
			// The first reconcile invocation should delete the regular DaemonSets/Deployments.
			t.AssertReconcileRequeue(ctx)

			t.AssertNoDaemonSet(ctx, names.GatewayComponent)
			t.AssertNoDaemonSet(ctx, names.RouteAgentComponent)
			t.AssertNoDaemonSet(ctx, names.GlobalnetComponent)

			// Simulate the gateway DaemonSet controller cleaning up its pods.
			t.DeletePods(ctx, "app", names.GatewayComponent)

			// Next, the controller should create the corresponding uninstall DaemonSets/Deployments.
			t.AssertReconcileRequeue(ctx)

			// For the globalnet DaemonSet, we'll only update it to nodes available but not yet ready at this point.
			globalnetDS := t.assertUninstallGlobalnetDaemonSet(ctx)
			t.UpdateDaemonSetToScheduled(ctx, globalnetDS)

			// For the gateway DaemonSet, we'll update it to observed but no nodes available - this will cause it to be deleted.
			t.UpdateDaemonSetToObserved(ctx, t.assertUninstallGatewayDaemonSet(ctx))

			t.UpdateDaemonSetToReady(ctx, t.assertUninstallRouteAgentDaemonSet(ctx))

			// Next, the controller should again requeue b/c the gateway DaemonSet isn't ready yet.
			t.AssertReconcileRequeue(ctx)

			// Now update the globalnet DaemonSet to ready.
			t.UpdateDaemonSetToReady(ctx, globalnetDS)

			// Ensure the finalizer is still present.
			t.awaitFinalizer()

			// Finally, the controller should delete the uninstall DaemonSets/Deployments and remove the finalizer.
			t.AssertReconcileSuccess(ctx)

			t.AssertNoDaemonSet(ctx, opnames.AppendUninstall(names.GatewayComponent))
			t.AssertNoDaemonSet(ctx, opnames.AppendUninstall(names.RouteAgentComponent))
			t.AssertNoDaemonSet(ctx, opnames.AppendUninstall(names.GlobalnetComponent))

			t.awaitSubmarinerDeleted()

			t.AssertReconcileSuccess(ctx)
			t.AssertNoDaemonSet(ctx, opnames.AppendUninstall(names.GatewayComponent))
		})
	})

	Context("and some components aren't installed", func() {
		BeforeEach(func() {
			t.submariner.Spec.GlobalCIDR = ""
			t.submariner.Spec.Version = "devel"

			t.InitScopedClientObjs = append(t.InitScopedClientObjs,
				t.NewDaemonSet(names.GatewayComponent),
				t.NewDaemonSet(names.RouteAgentComponent))
		})

		It("should only create uninstall DaemonSets/Deployments for installed components", func(ctx SpecContext) {
			t.AssertReconcileRequeue(ctx)

			t.UpdateDaemonSetToReady(ctx, t.assertUninstallGatewayDaemonSet(ctx))
			t.UpdateDaemonSetToReady(ctx, t.assertUninstallRouteAgentDaemonSet(ctx))

			t.AssertNoDaemonSet(ctx, opnames.AppendUninstall(names.GlobalnetComponent))

			t.AssertReconcileSuccess(ctx)

			t.AssertNoDaemonSet(ctx, opnames.AppendUninstall(names.GatewayComponent))
			t.AssertNoDaemonSet(ctx, opnames.AppendUninstall(names.RouteAgentComponent))

			t.awaitSubmarinerDeleted()
		})
	})

	Context("and an uninstall DaemonSet does not complete in time", func() {
		BeforeEach(func() {
			t.submariner.Spec.GlobalCIDR = ""

			t.InterceptorFuncs.Get = func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object,
				opts ...client.GetOption,
			) error {
				err := c.Get(ctx, key, obj, opts...)
				if err != nil {
					return err
				}

				if _, ok := obj.(*v1alpha1.Submariner); ok {
					obj.SetDeletionTimestamp(t.submariner.GetDeletionTimestamp())
				}

				return nil
			}

			t.InterceptorFuncs.Update = func(ctx context.Context, c client.WithWatch, obj client.Object,
				opts ...client.UpdateOption,
			) error {
				if _, ok := obj.(*v1alpha1.Submariner); ok {
					obj.SetDeletionTimestamp(&deletionTimestamp)
				}

				return c.Update(ctx, obj, opts...)
			}
		})

		It("should delete it", func(ctx SpecContext) {
			t.AssertReconcileRequeue(ctx)

			t.UpdateDaemonSetToReady(ctx, t.assertUninstallGatewayDaemonSet(ctx))
			t.UpdateDaemonSetToScheduled(ctx, t.assertUninstallRouteAgentDaemonSet(ctx))

			ts := metav1.NewTime(time.Now().Add(-(uninstall.ComponentReadyTimeout + 10)))
			t.submariner.SetDeletionTimestamp(&ts)

			t.AssertReconcileSuccess(ctx)

			t.AssertNoDaemonSet(ctx, opnames.AppendUninstall(names.GatewayComponent))
			t.AssertNoDaemonSet(ctx, opnames.AppendUninstall(names.RouteAgentComponent))

			t.awaitSubmarinerDeleted()
		})
	})

	Context("and the version of the deleting Submariner instance does not support uninstall", func() {
		BeforeEach(func() {
			t.submariner.Spec.Version = "0.11.1"

			t.InitScopedClientObjs = append(t.InitScopedClientObjs,
				t.NewDaemonSet(names.GatewayComponent))
		})

		It("should not perform uninstall", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)

			_, err := t.GetDaemonSet(ctx, names.GatewayComponent)
			Expect(err).To(Succeed())

			t.AssertNoDaemonSet(ctx, opnames.AppendUninstall(names.GatewayComponent))

			t.awaitSubmarinerDeleted()
		})
	})

	Context("and ServiceDiscovery is enabled", func() {
		BeforeEach(func() {
			t.submariner.Spec.GlobalCIDR = ""
			t.submariner.Spec.ServiceDiscoveryEnabled = true

			t.InitScopedClientObjs = append(t.InitScopedClientObjs,
				&v1alpha1.ServiceDiscovery{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: submarinerNamespace,
						Name:      opnames.ServiceDiscoveryCrName,
					},
				})
		})

		It("should delete the ServiceDiscovery resource", func(ctx SpecContext) {
			t.AssertReconcileRequeue(ctx)

			t.UpdateDaemonSetToReady(ctx, t.assertUninstallGatewayDaemonSet(ctx))
			t.UpdateDaemonSetToReady(ctx, t.assertUninstallRouteAgentDaemonSet(ctx))

			t.AssertReconcileSuccess(ctx)

			serviceDiscovery := &v1alpha1.ServiceDiscovery{}
			err := t.ScopedClient.Get(ctx, types.NamespacedName{Name: opnames.ServiceDiscoveryCrName, Namespace: submarinerNamespace},
				serviceDiscovery)
			Expect(errors.IsNotFound(err)).To(BeTrue(), "ServiceDiscovery still exists")

			t.awaitSubmarinerDeleted()
		})
	})
}

func newInfrastructureCluster(platformType v1config.PlatformType) *v1config.Infrastructure {
	return &v1config.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Status: v1config.InfrastructureStatus{
			PlatformStatus: &v1config.PlatformStatus{
				Type: platformType,
			},
		},
	}
}
