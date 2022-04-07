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
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/log/kzerolog"
	"github.com/submariner-io/admiral/pkg/syncer/broker"
	operatorv1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/constants"
	submarinerController "github.com/submariner-io/submariner-operator/controllers/submariner"
	"github.com/submariner-io/submariner-operator/controllers/test"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/names"
	submarinerv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
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

type testDriver struct {
	test.Driver
	submariner     *operatorv1.Submariner
	clusterNetwork *network.ClusterNetwork
}

func newTestDriver() *testDriver {
	t := &testDriver{
		Driver: test.Driver{
			Namespace:    submarinerNamespace,
			ResourceName: submarinerName,
		},
	}

	BeforeEach(func() {
		t.BeforeEach()
		t.submariner = newSubmariner()
		t.InitClientObjs = []controllerClient.Object{t.submariner}

		t.clusterNetwork = &network.ClusterNetwork{
			NetworkPlugin: "fake",
			ServiceCIDRs:  []string{testDetectedServiceCIDR},
			PodCIDRs:      []string{testDetectedClusterCIDR},
		}
	})

	JustBeforeEach(func() {
		t.JustBeforeEach()

		t.Controller = submarinerController.NewReconciler(&submarinerController.Config{
			Client:         t.Client,
			Scheme:         scheme.Scheme,
			ClusterNetwork: t.clusterNetwork,
		})
	})

	return t
}

func (t *testDriver) awaitFinalizer() {
	t.AwaitFinalizer(t.submariner, constants.CleanupFinalizer)
}

func (t *testDriver) awaitSubmarinerDeleted() {
	t.AwaitNoResource(t.submariner)
}

func (t *testDriver) getSubmariner() *operatorv1.Submariner {
	obj := &operatorv1.Submariner{}
	err := t.Client.Get(context.TODO(), types.NamespacedName{Name: submarinerName, Namespace: submarinerNamespace}, obj)
	Expect(err).To(Succeed())

	return obj
}

func (t *testDriver) assertRouteAgentDaemonSet() {
	daemonSet := t.AssertDaemonSet(names.RouteAgentComponent)

	Expect(daemonSet.Spec.Template.Spec.Containers).To(HaveLen(1))
	Expect(daemonSet.Spec.Template.Spec.Containers[0].Image).To(
		Equal(fmt.Sprintf("%s/%s:%s", t.submariner.Spec.Repository, names.RouteAgentImage, t.submariner.Spec.Version)))

	t.assertRouteAgentDaemonSetEnv(t.withNetworkDiscovery(), test.EnvMapFrom(daemonSet))
}

func (t *testDriver) assertUninstallRouteAgentDaemonSet() *appsv1.DaemonSet {
	daemonSet := t.AssertDaemonSet(names.AppendUninstall(names.RouteAgentComponent))

	envMap := t.AssertUninstallInitContainer(&daemonSet.Spec.Template,
		fmt.Sprintf("%s/%s:%s", t.submariner.Spec.Repository, names.RouteAgentImage, t.submariner.Spec.Version))
	t.assertRouteAgentDaemonSetEnv(t.withNetworkDiscovery(), envMap)

	return daemonSet
}

func (t *testDriver) assertRouteAgentDaemonSetEnv(submariner *operatorv1.Submariner, envMap map[string]string) {
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_NAMESPACE", submariner.Spec.Namespace))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_CLUSTERID", submariner.Spec.ClusterID))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_CLUSTERCIDR", submariner.Status.ClusterCIDR))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_SERVICECIDR", submariner.Status.ServiceCIDR))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_NETWORKPLUGIN", submariner.Status.NetworkPlugin))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_DEBUG", strconv.FormatBool(submariner.Spec.Debug)))
}

func (t *testDriver) assertGatewayDaemonSet() {
	daemonSet := t.AssertDaemonSet(names.GatewayComponent)
	assertGatewayNodeSelector(daemonSet)

	Expect(daemonSet.Spec.Template.Spec.Containers).To(HaveLen(1))
	Expect(daemonSet.Spec.Template.Spec.Containers[0].Image).To(
		Equal(fmt.Sprintf("%s/%s:%s", t.submariner.Spec.Repository, names.GatewayImage, t.submariner.Spec.Version)))

	t.assertGatewayDaemonSetEnv(t.withNetworkDiscovery(), test.EnvMapFrom(daemonSet))
}

func (t *testDriver) assertUninstallGatewayDaemonSet() *appsv1.DaemonSet {
	daemonSet := t.AssertDaemonSet(names.AppendUninstall(names.GatewayComponent))
	assertGatewayNodeSelector(daemonSet)

	envMap := t.AssertUninstallInitContainer(&daemonSet.Spec.Template,
		fmt.Sprintf("%s/%s:%s", t.submariner.Spec.Repository, names.GatewayImage, t.submariner.Spec.Version))
	t.assertGatewayDaemonSetEnv(t.withNetworkDiscovery(), envMap)

	return daemonSet
}

func (t *testDriver) assertGatewayDaemonSetEnv(submariner *operatorv1.Submariner, envMap map[string]string) {
	Expect(envMap).To(HaveKeyWithValue("CE_IPSEC_PSK", submariner.Spec.CeIPSecPSK))
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
}

func (t *testDriver) assertGlobalnetDaemonSet() {
	daemonSet := t.AssertDaemonSet(names.GlobalnetComponent)
	assertGatewayNodeSelector(daemonSet)

	Expect(daemonSet.Spec.Template.Spec.Containers).To(HaveLen(1))
	Expect(daemonSet.Spec.Template.Spec.Containers[0].Image).To(
		Equal(fmt.Sprintf("%s/%s:%s", t.submariner.Spec.Repository, names.GlobalnetImage, t.submariner.Spec.Version)))

	t.assertGlobalnetDaemonSetEnv(t.withNetworkDiscovery(), test.EnvMapFrom(daemonSet))
}

func (t *testDriver) assertUninstallGlobalnetDaemonSet() *appsv1.DaemonSet {
	daemonSet := t.AssertDaemonSet(names.AppendUninstall(names.GlobalnetComponent))
	assertGatewayNodeSelector(daemonSet)

	envMap := t.AssertUninstallInitContainer(&daemonSet.Spec.Template,
		fmt.Sprintf("%s/%s:%s", t.submariner.Spec.Repository, names.GlobalnetImage, t.submariner.Spec.Version))
	t.assertGlobalnetDaemonSetEnv(t.withNetworkDiscovery(), envMap)

	return daemonSet
}

func (t *testDriver) assertGlobalnetDaemonSetEnv(submariner *operatorv1.Submariner, envMap map[string]string) {
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_NAMESPACE", submariner.Spec.Namespace))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_CLUSTERID", submariner.Spec.ClusterID))
}

func (t *testDriver) assertNetworkPluginSyncerDeployment() {
	deployment := t.AssertDeployment(names.NetworkPluginSyncerComponent)

	Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))
	Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(
		Equal(fmt.Sprintf("%s/%s:%s", t.submariner.Spec.Repository, names.NetworkPluginSyncerImage, t.submariner.Spec.Version)))

	t.assertNetworkPluginSyncerDeploymentEnv(t.withNetworkDiscovery(),
		test.EnvMapFromVars(deployment.Spec.Template.Spec.Containers[0].Env))
}

func (t *testDriver) assertUninstallNetworkPluginSyncerDeployment() *appsv1.Deployment {
	deployment := t.AssertDeployment(names.AppendUninstall(names.NetworkPluginSyncerComponent))

	envMap := t.AssertUninstallInitContainer(&deployment.Spec.Template,
		fmt.Sprintf("%s/%s:%s", t.submariner.Spec.Repository, names.NetworkPluginSyncerImage, t.submariner.Spec.Version))
	t.assertNetworkPluginSyncerDeploymentEnv(t.withNetworkDiscovery(), envMap)

	return deployment
}

func (t *testDriver) assertNetworkPluginSyncerDeploymentEnv(submariner *operatorv1.Submariner, envMap map[string]string) {
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_NAMESPACE", submariner.Spec.Namespace))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_CLUSTERID", submariner.Spec.ClusterID))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_CLUSTERCIDR", submariner.Status.ClusterCIDR))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_SERVICECIDR", submariner.Status.ServiceCIDR))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_GLOBALCIDR", submariner.Status.GlobalCIDR))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_NETWORKPLUGIN", submariner.Status.NetworkPlugin))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_DEBUG", strconv.FormatBool(submariner.Spec.Debug)))
}

func assertGatewayNodeSelector(daemonSet *appsv1.DaemonSet) {
	Expect(daemonSet.Spec.Template.Spec.NodeSelector["submariner.io/gateway"]).To(Equal("true"))
}

func (t *testDriver) withNetworkDiscovery() *operatorv1.Submariner {
	t.submariner.Status.ClusterCIDR = getClusterCIDR(t.submariner, t.clusterNetwork)
	t.submariner.Status.ServiceCIDR = getServiceCIDR(t.submariner, t.clusterNetwork)
	t.submariner.Status.GlobalCIDR = getGlobalCIDR(t.submariner, t.clusterNetwork)
	t.submariner.Status.NetworkPlugin = t.clusterNetwork.NetworkPlugin

	return t.submariner
}

func newSubmariner() *operatorv1.Submariner {
	return &operatorv1.Submariner{
		ObjectMeta: metav1.ObjectMeta{
			Name:      submarinerName,
			Namespace: submarinerNamespace,
		},
		Spec: operatorv1.SubmarinerSpec{
			Repository:               "quay.io/submariner",
			Version:                  "0.12.0",
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

func getGlobalCIDR(submariner *operatorv1.Submariner, clusterNetwork *network.ClusterNetwork) string {
	if submariner.Spec.GlobalCIDR != "" {
		return submariner.Spec.GlobalCIDR
	}

	return clusterNetwork.GlobalCIDR
}
