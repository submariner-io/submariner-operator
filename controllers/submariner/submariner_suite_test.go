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
	"fmt"
	"reflect"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/log/kzerolog"
	"github.com/submariner-io/admiral/pkg/syncer/broker"
	operatorv1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	submarinerController "github.com/submariner-io/submariner-operator/controllers/submariner"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/names"
	submarinerv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes/scheme"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = BeforeSuite(func() {
	Expect(operatorv1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(apiextensions.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(submarinerv1.AddToScheme(scheme.Scheme)).To(Succeed())
})

var _ = Describe("", func() {
	kzerolog.InitK8sLogging()
})

func TestSubmariner(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Submariner Suite")
}

type failingClient struct {
	controllerClient.Client
	onCreate reflect.Type
	onGet    reflect.Type
	onUpdate reflect.Type
}

func (c *failingClient) Create(ctx context.Context, obj controllerClient.Object, opts ...controllerClient.CreateOption) error {
	if c.onCreate == reflect.TypeOf(obj) {
		return fmt.Errorf("Mock Create error")
	}

	return c.Client.Create(ctx, obj, opts...)
}

func (c *failingClient) Get(ctx context.Context, key controllerClient.ObjectKey, obj controllerClient.Object) error {
	if c.onGet == reflect.TypeOf(obj) {
		return fmt.Errorf("Mock Get error")
	}

	return c.Client.Get(ctx, key, obj)
}

func (c *failingClient) Update(ctx context.Context, obj controllerClient.Object, opts ...controllerClient.UpdateOption) error {
	if c.onUpdate == reflect.TypeOf(obj) {
		return fmt.Errorf("Mock Update error")
	}

	return c.Client.Update(ctx, obj, opts...)
}

type testDriver struct {
	initClientObjs []controllerClient.Object
	fakeClient     controllerClient.Client
	submariner     *operatorv1.Submariner
	controller     *submarinerController.Reconciler
	clusterNetwork *network.ClusterNetwork
}

func newTestDriver() *testDriver {
	t := &testDriver{}

	BeforeEach(func() {
		t.fakeClient = nil
		t.submariner = newSubmariner()
		t.initClientObjs = []controllerClient.Object{t.submariner}

		t.clusterNetwork = &network.ClusterNetwork{
			NetworkPlugin: "fake",
			ServiceCIDRs:  []string{testDetectedServiceCIDR},
			PodCIDRs:      []string{testDetectedClusterCIDR},
		}
	})

	JustBeforeEach(func() {
		if t.fakeClient == nil {
			t.fakeClient = t.newClient()
		}

		t.controller = submarinerController.NewReconciler(&submarinerController.Config{
			Client:         t.fakeClient,
			Scheme:         scheme.Scheme,
			ClusterNetwork: t.clusterNetwork,
		})
	})

	return t
}

func (t *testDriver) newClient() controllerClient.Client {
	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(t.initClientObjs...).Build()
}

func (t *testDriver) doReconcile() (reconcile.Result, error) {
	return t.controller.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: submarinerNamespace,
		Name:      submarinerName,
	}})
}

func (t *testDriver) assertReconcileSuccess() {
	r, err := t.doReconcile()
	Expect(err).To(Succeed())
	Expect(r.Requeue).To(BeFalse())
	Expect(r.RequeueAfter).To(BeNumerically("==", 0))
}

func (t *testDriver) assertReconcileRequeue() {
	r, err := t.doReconcile()
	Expect(err).To(Succeed())
	Expect(r.RequeueAfter).To(BeNumerically(">", 0), "Expected requeue after")
	Expect(r.Requeue).To(BeFalse())
}

func (t *testDriver) getSubmariner() *operatorv1.Submariner {
	obj := &operatorv1.Submariner{}
	err := t.fakeClient.Get(context.TODO(), types.NamespacedName{Name: submarinerName, Namespace: submarinerNamespace}, obj)
	Expect(err).To(Succeed())

	return obj
}

func (t *testDriver) assertRouteAgentDaemonSet(submariner *operatorv1.Submariner) {
	daemonSet := t.assertDaemonSet(routeAgentDaemonSetName)

	Expect(daemonSet.ObjectMeta.Labels["app"]).To(Equal("submariner-routeagent"))
	Expect(daemonSet.Spec.Selector).To(Equal(&metav1.LabelSelector{MatchLabels: map[string]string{"app": "submariner-routeagent"}}))
	Expect(daemonSet.Spec.Template.ObjectMeta.Labels["app"]).To(Equal("submariner-routeagent"))
	Expect(daemonSet.Spec.Template.Spec.Containers).To(HaveLen(1))
	Expect(daemonSet.Spec.Template.Spec.Containers[0].Image).To(Equal(submariner.Spec.Repository + "/submariner-route-agent:" +
		submariner.Spec.Version))

	envMap := map[string]string{}
	for _, envVar := range daemonSet.Spec.Template.Spec.Containers[0].Env {
		envMap[envVar.Name] = envVar.Value
	}

	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_NAMESPACE", submariner.Spec.Namespace))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_CLUSTERID", submariner.Spec.ClusterID))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_CLUSTERCIDR", submariner.Status.ClusterCIDR))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_SERVICECIDR", submariner.Status.ServiceCIDR))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_NETWORKPLUGIN", "fake"))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_DEBUG", strconv.FormatBool(submariner.Spec.Debug)))
}

func (t *testDriver) assertGatewayDaemonSet() {
	daemonSet := t.assertDaemonSet(names.GatewayComponent)
	assertGatewayNodeSelector(daemonSet)

	Expect(daemonSet.Spec.Template.Spec.Containers).To(HaveLen(1))
	Expect(daemonSet.Spec.Template.Spec.Containers[0].Image).To(
		Equal(fmt.Sprintf("%s/%s:%s", t.submariner.Spec.Repository, names.GatewayImage, t.submariner.Spec.Version)))

	t.assertGatewayDaemonSetEnv(t.withNetworkDiscovery(), daemonSet.Spec.Template.Spec.Containers[0].Env)
}

func (t *testDriver) assertUninstallGatewayDaemonSet() *appsv1.DaemonSet {
	daemonSet := t.assertDaemonSet(names.AppendUninstall(names.GatewayComponent))
	assertGatewayNodeSelector(daemonSet)

	Expect(daemonSet.Spec.Template.Spec.InitContainers).To(HaveLen(1))
	Expect(daemonSet.Spec.Template.Spec.InitContainers[0].Image).To(
		Equal(fmt.Sprintf("%s/%s:%s", t.submariner.Spec.Repository, names.GatewayImage, t.submariner.Spec.Version)))

	envMap := t.assertGatewayDaemonSetEnv(t.withNetworkDiscovery(), daemonSet.Spec.Template.Spec.InitContainers[0].Env)
	Expect(envMap).To(HaveKeyWithValue("UNINSTALL", "true"))

	return daemonSet
}

func (t *testDriver) assertGatewayDaemonSetEnv(submariner *operatorv1.Submariner, env []corev1.EnvVar) map[string]string {
	envMap := envMapFromVars(env)

	Expect(envMap).To(HaveKeyWithValue("CE_IPSEC_PSK", submariner.Spec.CeIPSecPSK))
	Expect(envMap).To(HaveKeyWithValue("CE_IPSEC_IKEPORT", strconv.Itoa(submariner.Spec.CeIPSecIKEPort)))
	Expect(envMap).To(HaveKeyWithValue("CE_IPSEC_NATTPORT", strconv.Itoa(submariner.Spec.CeIPSecNATTPort)))
	Expect(envMap).To(HaveKeyWithValue(broker.EnvironmentVariable("RemoteNamespace"), submariner.Spec.BrokerK8sRemoteNamespace))
	Expect(envMap).To(HaveKeyWithValue(broker.EnvironmentVariable("ApiServer"), submariner.Spec.BrokerK8sApiServer))
	Expect(envMap).To(HaveKeyWithValue(broker.EnvironmentVariable("ApiServerToken"), submariner.Spec.BrokerK8sApiServerToken))
	Expect(envMap).To(HaveKeyWithValue(broker.EnvironmentVariable("CA"), submariner.Spec.BrokerK8sCA))
	Expect(envMap).To(HaveKeyWithValue(broker.EnvironmentVariable("Insecure"), strconv.FormatBool(submariner.Spec.BrokerK8sInsecure)))
	Expect(envMap).To(HaveKeyWithValue(broker.EnvironmentVariable("Secret"), submariner.Spec.BrokerK8sSecret))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_BROKER", submariner.Spec.Broker))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_NATENABLED", strconv.FormatBool(submariner.Spec.
		NatEnabled)))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_CLUSTERID", submariner.Spec.ClusterID))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_SERVICECIDR", submariner.Status.ServiceCIDR))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_CLUSTERCIDR", submariner.Status.ClusterCIDR))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_GLOBALCIDR", submariner.Spec.GlobalCIDR))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_NAMESPACE", submariner.Spec.Namespace))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_DEBUG", strconv.FormatBool(submariner.Spec.Debug)))

	return envMap
}

func (t *testDriver) getDaemonSet(name string) (*appsv1.DaemonSet, error) {
	foundDaemonSet := &appsv1.DaemonSet{}
	err := t.fakeClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: submarinerNamespace}, foundDaemonSet)

	return foundDaemonSet, err
}

func (t *testDriver) assertDaemonSet(name string) *appsv1.DaemonSet {
	daemonSet, err := t.getDaemonSet(name)
	Expect(err).To(Succeed())

	Expect(daemonSet.ObjectMeta.Labels).To(HaveKeyWithValue("app", name))
	Expect(daemonSet.Spec.Selector.MatchLabels).To(HaveKeyWithValue("app", name))

	for k, v := range daemonSet.Spec.Selector.MatchLabels {
		Expect(daemonSet.Spec.Template.ObjectMeta.Labels).To(HaveKeyWithValue(k, v))
	}

	return daemonSet
}

func (t *testDriver) assertNoDaemonSet(name string) {
	_, err := t.getDaemonSet(name)
	Expect(errors.IsNotFound(err)).To(BeTrue(), "IsNotFound error")
	Expect(err).To(HaveOccurred())
}

func assertGatewayNodeSelector(daemonSet *appsv1.DaemonSet) {
	Expect(daemonSet.Spec.Template.Spec.NodeSelector["submariner.io/gateway"]).To(Equal("true"))
}

func (t *testDriver) updateDaemonSetToReady(daemonSet *appsv1.DaemonSet) {
	daemonSet.Status.NumberReady = daemonSet.Status.DesiredNumberScheduled

	Expect(t.fakeClient.Update(context.TODO(), daemonSet)).To(Succeed())
}

func (t *testDriver) updateDaemonSetToObserved(daemonSet *appsv1.DaemonSet) {
	daemonSet.Generation = 1
	daemonSet.Status.ObservedGeneration = 1
	daemonSet.Status.DesiredNumberScheduled = 1

	Expect(t.fakeClient.Update(context.TODO(), daemonSet)).To(Succeed())
}

func (t *testDriver) withNetworkDiscovery() *operatorv1.Submariner {
	t.submariner.Status.ClusterCIDR = getClusterCIDR(t.submariner, t.clusterNetwork)
	t.submariner.Status.ServiceCIDR = getServiceCIDR(t.submariner, t.clusterNetwork)
	t.submariner.Status.NetworkPlugin = t.clusterNetwork.NetworkPlugin

	return t.submariner
}

func (t *testDriver) newDaemonSet(name string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.submariner.Namespace,
			Name:      name,
		},
	}
}

func (t *testDriver) newPodWithLabel(label, value string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.submariner.Namespace,
			Name:      string(uuid.NewUUID()),
			Labels: map[string]string{
				label: value,
			},
		},
	}
}

func (t *testDriver) deletePods(label, value string) {
	err := t.fakeClient.DeleteAllOf(context.TODO(), &corev1.Pod{}, controllerClient.InNamespace(t.submariner.Namespace),
		controllerClient.MatchingLabelsSelector{Selector: labels.SelectorFromSet(map[string]string{label: value})})
	Expect(err).To(Succeed())
}

func envMapFrom(daemonSet *appsv1.DaemonSet) map[string]string {
	return envMapFromVars(daemonSet.Spec.Template.Spec.Containers[0].Env)
}

func envMapFromVars(env []corev1.EnvVar) map[string]string {
	envMap := map[string]string{}
	for _, envVar := range env {
		envMap[envVar.Name] = envVar.Value
	}

	return envMap
}

func newSubmariner() *operatorv1.Submariner {
	return &operatorv1.Submariner{
		ObjectMeta: metav1.ObjectMeta{
			Name:      submarinerName,
			Namespace: submarinerNamespace,
		},
		Spec: operatorv1.SubmarinerSpec{
			Repository:               "quay.io/submariner",
			Version:                  "1.0.0",
			CeIPSecNATTPort:          4500,
			CeIPSecIKEPort:           500,
			CeIPSecPSK:               "DJaA2kVW72w8kjQCEpzkDhwZuniDwgePKFE7FaxVNMWqbpmT2qvp68XW52MO70ho",
			BrokerK8sRemoteNamespace: "submariner-broker",
			BrokerK8sApiServer:       "https://192.168.99.110:8443",
			BrokerK8sApiServerToken:  "MIIDADCCAeigAw",
			BrokerK8sCA:              "client.crt",
			Broker:                   "k8s",
			NatEnabled:               true,
			ClusterID:                "east",
			ServiceCIDR:              "",
			ClusterCIDR:              "",
			GlobalCIDR:               "169.254.0.0/16",
			ColorCodes:               "red",
			Namespace:                "submariner_ns",
			Debug:                    true,
		},
	}
}

func getClusterCIDR(submariner *operatorv1.Submariner, clusterNetwork *network.ClusterNetwork) string {
	if submariner.Spec.ClusterCIDR != "" {
		return submariner.Spec.ClusterCIDR
	}

	return clusterNetwork.PodCIDRs[0]
}

func getServiceCIDR(submariner *operatorv1.Submariner, clusterNetwork *network.ClusterNetwork) string {
	if submariner.Spec.ServiceCIDR != "" {
		return submariner.Spec.ServiceCIDR
	}

	return clusterNetwork.ServiceCIDRs[0]
}
