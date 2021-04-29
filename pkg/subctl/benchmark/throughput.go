/*
© 2021 Red Hat, Inc. and others

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
package benchmark

import (
	"fmt"

	"github.com/onsi/gomega"
	"github.com/submariner-io/shipyard/test/e2e/framework"
	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var Verbose bool

func StartThroughputTests(intraCluster bool) {
	var f *framework.Framework

	gomega.RegisterFailHandler(func(message string, callerSkip ...int) {
		if f != nil {
			cleanupFramework(f)
		} else {
			framework.RunCleanupActions()
		}
		panic(message)
	})

	f = initFramework("throughput")

	clusterAName := framework.TestContext.ClusterIDs[framework.ClusterA]

	if !intraCluster {
		testParams := benchmarkTestParams{
			ClientCluster:       framework.ClusterA,
			ServerCluster:       framework.ClusterB,
			ServerPodScheduling: framework.GatewayNode,
			ClientPodScheduling: framework.GatewayNode,
		}

		clusterBName := framework.TestContext.ClusterIDs[framework.ClusterB]
		framework.By(fmt.Sprintf("Performing throughput tests from Gateway pod on cluster %q to Gateway pod on cluster %q",
			clusterAName, clusterBName))
		runThroughputTest(f, testParams)

		testParams.ServerPodScheduling = framework.NonGatewayNode
		testParams.ClientPodScheduling = framework.NonGatewayNode

		framework.By(fmt.Sprintf("Performing throughput tests from Non-Gateway pod on cluster %q to Non-Gateway pod on cluster %q",
			clusterAName, clusterBName))
		runThroughputTest(f, testParams)
	} else {
		testIntraClusterParams := benchmarkTestParams{
			ClientCluster:       framework.ClusterA,
			ServerCluster:       framework.ClusterA,
			ServerPodScheduling: framework.GatewayNode,
			ClientPodScheduling: framework.NonGatewayNode,
		}

		framework.By(fmt.Sprintf("Performing throughput tests from Non-Gateway pod to Gateway pod on cluster %q", clusterAName))
		runThroughputTest(f, testIntraClusterParams)
	}

	cleanupFramework(f)
}

var By = func(str string) {
	if Verbose {
		fmt.Println(str)
	}
}

func initFramework(baseName string) *framework.Framework {
	f := framework.NewBareFramework(baseName)
	framework.SetStatusFunction(By)

	framework.ValidateFlags(framework.TestContext)
	framework.BeforeSuite()
	f.BeforeEach()
	return f
}

func cleanupFramework(f *framework.Framework) {
	f.AfterEach()
	framework.RunCleanupActions()
}

func runThroughputTest(f *framework.Framework, testParams benchmarkTestParams) {
	clientClusterName := framework.TestContext.ClusterIDs[testParams.ClientCluster]
	serverClusterName := framework.TestContext.ClusterIDs[testParams.ServerCluster]
	var connectionTimeout uint = 10
	var connectionAttempts uint = 2
	var iperf3Port = 5201

	framework.By(fmt.Sprintf("Creating a Nettest Server Pod on %q", serverClusterName))
	nettestServerPod := f.NewNetworkPod(&framework.NetworkPodConfig{
		Type:               framework.ThroughputServerPod,
		Cluster:            testParams.ServerCluster,
		Scheduling:         testParams.ServerPodScheduling,
		ConnectionTimeout:  connectionTimeout,
		ConnectionAttempts: connectionAttempts,
		Port:               iperf3Port,
	})

	podsClusterB := framework.KubeClients[testParams.ServerCluster].CoreV1().Pods(f.Namespace)
	p1, _ := podsClusterB.Get(nettestServerPod.Pod.Name, metav1.GetOptions{})
	framework.By(fmt.Sprintf("Nettest Server Pod %q was created on node %q", nettestServerPod.Pod.Name, nettestServerPod.Pod.Spec.NodeName))

	remoteIP := p1.Status.PodIP

	var service *v1.Service
	if framework.TestContext.GlobalnetEnabled && testParams.ClientCluster != testParams.ServerCluster {
		By(fmt.Sprintf("Pointing a ClusterIP service to the nettest server pod in cluster %q and exporting it",
			framework.TestContext.ClusterIDs[testParams.ServerCluster]))
		service = nettestServerPod.CreateService()
		f.CreateServiceExport(testParams.ServerCluster, service.Name)

		// Wait for the globalIP annotation on the service.
		service = f.AwaitUntilAnnotationOnService(testParams.ServerCluster, globalnetGlobalIPAnnotation, service.Name, service.Namespace)
		remoteIP = service.GetAnnotations()[globalnetGlobalIPAnnotation]
	}

	nettestClientPod := f.NewNetworkPod(&framework.NetworkPodConfig{
		Type:               framework.ThroughputClientPod,
		Cluster:            testParams.ClientCluster,
		Scheduling:         testParams.ClientPodScheduling,
		RemoteIP:           remoteIP,
		ConnectionTimeout:  connectionTimeout,
		ConnectionAttempts: connectionAttempts,
		Port:               iperf3Port,
	})

	framework.By(fmt.Sprintf("Nettest Client Pod %q was created on cluster %q, node %q; connect to server pod ip %q",
		nettestClientPod.Pod.Name, clientClusterName, nettestClientPod.Pod.Spec.NodeName, remoteIP))

	framework.By(fmt.Sprintf("Waiting for the client pod %q to exit, returning what client sent", nettestClientPod.Pod.Name))
	nettestClientPod.AwaitFinishVerbose(Verbose)
	nettestClientPod.CheckSuccessfulFinish()
	fmt.Println(nettestClientPod.TerminationMessage)

	// In Globalnet deployments, when backend pods finish their execution, kubeproxy-iptables driver tries
	// to delete the iptables-chain associated with the service (even when the service is present) as there are
	// no active backend pods. Since the iptables-chain is also referenced by Globalnet Ingress rules, the chain
	// cannot be deleted (kubeproxy errors out and continues to retry) until Globalnet removes the reference.
	// Globalnet removes the reference only when the service itself is deleted. Until Globalnet is enhanced [*]
	// to remove this dependency with iptables-chain, lets delete the service after the nettest server Pod is terminated.
	// [*] https://github.com/submariner-io/submariner/issues/1166
	if framework.TestContext.GlobalnetEnabled && testParams.ClientCluster != testParams.ServerCluster {
		f.DeletePod(testParams.ServerCluster, nettestServerPod.Pod.Name, f.Namespace)
		f.DeleteService(testParams.ServerCluster, service.Name)
		f.DeleteServiceExport(testParams.ServerCluster, service.Name)
	}
}
