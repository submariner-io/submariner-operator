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

package cleanup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/watcher"
	"github.com/submariner-io/shipyard/test/e2e/framework"
	operatorv1alpha1 "github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/uninstall"
	"github.com/submariner-io/submariner-operator/pkg/names"
	submarinerclientset "github.com/submariner-io/submariner/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Submariner cleanup", func() {
	When("the Submariner resource is deleted", testSubmarinerCleanup)
})

func testSubmarinerCleanup() {
	var (
		crClient                      client.Client
		submariner                    *operatorv1alpha1.Submariner
		routeAgentPodMonitor          *podMonitor
		gatewayPodMonitor             *podMonitor
		globalnetPodMonitor           *podMonitor
		networkPluginSyncerPodMonitor *podMonitor
		lhAgentAgentPodMonitor        *podMonitor
		brokerRestConfig              *rest.Config
		stopCh                        chan struct{}
	)

	BeforeEach(func() {
		stopCh = make(chan struct{})

		framework.DetectGlobalnet()

		var err error

		crClient, err = client.New(framework.RestConfigs[framework.ClusterA], client.Options{})
		Expect(err).To(Succeed())
		Expect(operatorv1alpha1.AddToScheme(crClient.Scheme())).To(Succeed())

		submariner = &operatorv1alpha1.Submariner{}
		err = crClient.Get(
			context.TODO(),
			client.ObjectKey{
				Namespace: framework.TestContext.SubmarinerNamespace,
				Name:      names.SubmarinerCrName,
			},
			submariner)
		Expect(err).To(Succeed())

		brokerRestConfig = getBrokerRestConfig(submariner.Spec.BrokerK8sRemoteNamespace)
		Expect(brokerRestConfig).ToNot(BeNil(), "No broker located")

		routeAgentPodMonitor = startPodMonitor(names.AppendUninstall(names.RouteAgentComponent), stopCh)
		gatewayPodMonitor = startPodMonitor(names.AppendUninstall(names.GatewayComponent), stopCh)

		if framework.TestContext.GlobalnetEnabled {
			globalnetPodMonitor = startPodMonitor(names.AppendUninstall(names.GlobalnetComponent), stopCh)
		}

		_, err = framework.KubeClients[framework.ClusterA].AppsV1().Deployments(framework.TestContext.SubmarinerNamespace).Get(context.TODO(),
			names.NetworkPluginSyncerComponent, metav1.GetOptions{})
		if err == nil {
			networkPluginSyncerPodMonitor = startPodMonitor(names.AppendUninstall(names.NetworkPluginSyncerComponent), stopCh)
		}

		if submariner.Spec.ServiceDiscoveryEnabled {
			lhAgentAgentPodMonitor = startPodMonitor(names.AppendUninstall(names.ServiceDiscoveryComponent), stopCh)
		}
	})

	AfterEach(func() {
		close(stopCh)

		if submariner != nil {
			submariner.ObjectMeta = metav1.ObjectMeta{
				Namespace:   submariner.Namespace,
				Name:        submariner.Name,
				Labels:      submariner.Labels,
				Annotations: submariner.Annotations,
			}

			err := crClient.Create(context.TODO(), submariner)
			if err != nil {
				framework.Errorf("Error re-creating Submariner: %v", err)
			} else {
				By("Re-created Submariner resource")
			}
		}
	})

	It("should run DaemonSets/Deployments to uninstall components", func() {
		By(fmt.Sprintf("Deleting Submariner resource %v in %q", submariner, framework.TestContext.ClusterIDs[framework.ClusterA]))

		err := crClient.Delete(context.TODO(), submariner)
		Expect(err).To(Succeed())

		Eventually(func() bool {
			err := crClient.Get(
				context.TODO(),
				client.ObjectKey{
					Namespace: framework.TestContext.SubmarinerNamespace,
					Name:      names.SubmarinerCrName,
				},
				submariner)
			return apierrors.IsNotFound(err)
		}, uninstall.ComponentReadyTimeout*2).Should(BeTrue(), "Submariner resource not deleted")

		By("Submariner resource deleted")

		serviceDiscovery := &operatorv1alpha1.ServiceDiscovery{}

		err = crClient.Get(
			context.TODO(),
			client.ObjectKey{
				Namespace: framework.TestContext.SubmarinerNamespace,
				Name:      names.ServiceDiscoveryCrName,
			},
			serviceDiscovery)
		assertIsNotFound(err, "ServiceDiscovery", names.ServiceDiscoveryCrName)

		lhAgentAgentPodMonitor.assertUninstallPodsCompleted()
		routeAgentPodMonitor.assertUninstallPodsCompleted()
		gatewayPodMonitor.assertUninstallPodsCompleted()
		globalnetPodMonitor.assertUninstallPodsCompleted()
		networkPluginSyncerPodMonitor.assertUninstallPodsCompleted()

		_, err = framework.KubeClients[framework.ClusterA].AppsV1().DaemonSets(framework.TestContext.SubmarinerNamespace).Get(context.TODO(),
			names.AppendUninstall(names.GatewayComponent), metav1.GetOptions{})
		assertIsNotFound(err, "DaemonSet", names.AppendUninstall(names.GatewayComponent))

		_, err = framework.KubeClients[framework.ClusterA].AppsV1().DaemonSets(framework.TestContext.SubmarinerNamespace).Get(context.TODO(),
			names.AppendUninstall(names.RouteAgentComponent), metav1.GetOptions{})
		assertIsNotFound(err, "DaemonSet", names.AppendUninstall(names.RouteAgentComponent))

		_, err = framework.KubeClients[framework.ClusterA].AppsV1().DaemonSets(framework.TestContext.SubmarinerNamespace).Get(context.TODO(),
			names.AppendUninstall(names.GlobalnetComponent), metav1.GetOptions{})
		assertIsNotFound(err, "DaemonSet", names.AppendUninstall(names.GlobalnetComponent))

		_, err = framework.KubeClients[framework.ClusterA].AppsV1().Deployments(framework.TestContext.SubmarinerNamespace).Get(context.TODO(),
			names.AppendUninstall(names.NetworkPluginSyncerComponent), metav1.GetOptions{})
		assertIsNotFound(err, "Deployment", names.AppendUninstall(names.NetworkPluginSyncerComponent))

		_, err = framework.KubeClients[framework.ClusterA].AppsV1().Deployments(framework.TestContext.SubmarinerNamespace).Get(context.TODO(),
			names.AppendUninstall(names.ServiceDiscoveryComponent), metav1.GetOptions{})
		assertIsNotFound(err, "Deployment", names.AppendUninstall(names.ServiceDiscoveryComponent))

		assertNoClusterResources(brokerRestConfig, submariner.Spec.BrokerK8sRemoteNamespace)
		assertNoEndpointResources(brokerRestConfig, submariner.Spec.BrokerK8sRemoteNamespace)
		assertNoGlobalnetResources()

		if submariner.Spec.ServiceDiscoveryEnabled {
			Expect(getCorednsConfigMap().Data["Corefile"]).ToNot(ContainSubstring("lighthouse"))
		}
	})
}

func assertNoGlobalnetResources() {
	submarinerClient, err := submarinerclientset.NewForConfig(framework.RestConfigs[framework.ClusterA])
	Expect(err).To(Succeed())

	list, err := submarinerClient.SubmarinerV1().ClusterGlobalEgressIPs(metav1.NamespaceNone).List(context.TODO(), metav1.ListOptions{})
	Expect(err).To(Succeed())
	Expect(list.Items).To(BeEmpty())
}

func assertNoEndpointResources(brokerRestConfig *rest.Config, brokerNS string) {
	clusterID := framework.TestContext.ClusterIDs[framework.ClusterA]

	By(fmt.Sprintf("Verifying no Endpoint resources for %q", clusterID))

	assertNoEndpoints := func(restConfig *rest.Config, namespace, msgFormat string) {
		client, err := submarinerclientset.NewForConfig(restConfig)
		Expect(err).To(Succeed())

		endpoints, err := client.SubmarinerV1().Endpoints(namespace).List(context.TODO(), metav1.ListOptions{})
		Expect(err).To(Succeed())

		for i := range endpoints.Items {
			Expect(endpoints.Items[i].Spec.ClusterID).ToNot(Equal(clusterID), fmt.Sprintf(msgFormat, endpoints.Items[i].Name))
		}
	}

	assertNoEndpoints(brokerRestConfig, brokerNS, "Unexpected Endpoint %q found on the broker")
	assertNoEndpoints(framework.RestConfigs[framework.ClusterA], framework.TestContext.SubmarinerNamespace,
		"Unexpected local Endpoint %q found")
}

func assertNoClusterResources(brokerRestConfig *rest.Config, brokerNS string) {
	clusterID := framework.TestContext.ClusterIDs[framework.ClusterA]

	By(fmt.Sprintf("Verifying no Cluster resources for %q", clusterID))

	assertNoCluster := func(restConfig *rest.Config, namespace, msgFormat string) {
		client, err := submarinerclientset.NewForConfig(restConfig)
		Expect(err).To(Succeed())

		_, err = client.SubmarinerV1().Clusters(namespace).Get(context.TODO(), clusterID, metav1.GetOptions{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), fmt.Sprintf(msgFormat, clusterID))
	}

	assertNoCluster(brokerRestConfig, brokerNS, "Unexpected Cluster %q found on the broker")
	assertNoCluster(framework.RestConfigs[framework.ClusterA], framework.TestContext.SubmarinerNamespace,
		"Unexpected local Cluster %q found")
}

type podInfo struct {
	status  corev1.PodStatus
	log     string
	prevLog string
}

type podMonitor struct {
	sync.Mutex
	name string
	pods map[string]*podInfo
}

func startPodMonitor(name string, stopCh <-chan struct{}) *podMonitor {
	m := &podMonitor{
		name: name,
		pods: map[string]*podInfo{},
	}

	w, err := watcher.New(&watcher.Config{
		RestConfig: framework.RestConfigs[framework.ClusterA],
		ResourceConfigs: []watcher.ResourceConfig{
			{
				Name:         m.name,
				ResourceType: &corev1.Pod{},
				Handler: watcher.EventHandlerFuncs{
					OnCreateFunc: m.processPod,
					OnUpdateFunc: m.processPod,
				},
				SourceNamespace:     framework.TestContext.SubmarinerNamespace,
				SourceLabelSelector: "app=" + name,
			},
		},
	})
	Expect(err).To(Succeed())

	err = w.Start(stopCh)
	Expect(err).To(Succeed())

	return m
}

func (m *podMonitor) processPod(obj runtime.Object, _ int) bool {
	defer GinkgoRecover()

	pod := obj.(*corev1.Pod)

	m.Lock()
	defer m.Unlock()

	info := m.pods[pod.Name]
	if info == nil {
		info = &podInfo{}
		m.pods[pod.Name] = info
	}

	info.status = pod.Status

	logOptions := corev1.PodLogOptions{
		Container: m.name,
	}

	getPodLog(pod.Name, &logOptions, &info.log)

	logOptions.Previous = true
	getPodLog(pod.Name, &logOptions, &info.prevLog)

	return false
}

func (m *podMonitor) assertUninstallPodsCompleted() {
	if m == nil {
		return
	}

	By(fmt.Sprintf("Verifying uninstall pods for component %q completed", m.name))

	m.Lock()
	defer m.Unlock()

	Expect(len(m.pods)).ToNot(BeZero(), fmt.Sprintf("No uninstall pods were created for component %q", m.name))

	for name, info := range m.pods {
		if info.status.Phase == corev1.PodRunning {
			continue
		}

		status, _ := json.MarshalIndent(info.status, "", "  ")

		log := info.prevLog
		if log == "" {
			log = info.log
		}

		Fail(fmt.Sprintf("Pod %q did not complete\nSTATUS: %s\n\nLOG\n: %s\n", name, string(status), log))
	}
}

func getPodLog(name string, options *corev1.PodLogOptions, writeTo *string) {
	req := framework.KubeClients[framework.ClusterA].CoreV1().Pods(framework.TestContext.SubmarinerNamespace).GetLogs(name, options)

	stream, err := req.Stream(context.TODO())
	if err != nil {
		return
	}

	defer stream.Close()

	log := new(bytes.Buffer)

	_, err = io.Copy(log, stream)
	if err != nil {
		return
	}

	*writeTo = log.String()
}

func getBrokerRestConfig(brokerNS string) *rest.Config {
	for i := range framework.RestConfigs {
		_, err := framework.KubeClients[i].CoreV1().Namespaces().Get(context.TODO(), brokerNS, metav1.GetOptions{})
		if err == nil {
			return framework.RestConfigs[i]
		}
	}

	return nil
}

func getCorednsConfigMap() *corev1.ConfigMap {
	configMap, err := framework.KubeClients[framework.ClusterA].CoreV1().ConfigMaps("kube-system").Get(context.TODO(),
		"coredns", metav1.GetOptions{})
	Expect(err).To(Succeed())

	return configMap
}

func assertIsNotFound(err error, typeName, objName string) {
	if apierrors.IsNotFound(err) {
		return
	}

	if err == nil {
		Fail(fmt.Sprintf("Unexpected %s %q found", typeName, objName))
	}

	Expect(err).To(Succeed())
}
