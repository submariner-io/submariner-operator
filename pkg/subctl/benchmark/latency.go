package benchmark

import (
	"fmt"

	"github.com/onsi/gomega"
	"github.com/submariner-io/shipyard/test/e2e/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const globalnetGlobalIPAnnotation = "submariner.io/globalIp"

type benchmarkTestParams struct {
	ClientCluster       framework.ClusterIndex
	ServerCluster       framework.ClusterIndex
	ServerPodScheduling framework.NetworkPodScheduling
	ClientPodScheduling framework.NetworkPodScheduling
}

func StartLatencyTests(intraCluster bool) {
	var f *framework.Framework

	gomega.RegisterFailHandler(func(message string, callerSkip ...int) {
		if f != nil {
			cleanupFramework(f)
		} else {
			framework.RunCleanupActions()
		}
		panic(message)
	})

	f = initFramework("latency")

	if !intraCluster {
		latencyTestParams := benchmarkTestParams{
			ClientCluster:       framework.ClusterA,
			ServerCluster:       framework.ClusterB,
			ServerPodScheduling: framework.GatewayNode,
			ClientPodScheduling: framework.GatewayNode,
		}

		framework.By("Performing latency tests from Gateway pod to Gateway pod")
		runLatencyTest(f, latencyTestParams)

		latencyTestParams.ServerPodScheduling = framework.NonGatewayNode
		latencyTestParams.ClientPodScheduling = framework.NonGatewayNode
		framework.By("Performing latency tests from Non-Gateway pod to Non-Gateway pod")
		runLatencyTest(f, latencyTestParams)
	} else {
		latencyTestIntraClusterParams := benchmarkTestParams{
			ClientCluster:       framework.ClusterA,
			ServerCluster:       framework.ClusterA,
			ServerPodScheduling: framework.GatewayNode,
			ClientPodScheduling: framework.NonGatewayNode,
		}

		clusterAName := framework.TestContext.ClusterIDs[framework.ClusterA]
		framework.By(fmt.Sprintf("Performing latency tests from Non-Gateway pod to Gateway pod on cluster %q", clusterAName))
		runLatencyTest(f, latencyTestIntraClusterParams)
	}

	cleanupFramework(f)
}

func runLatencyTest(f *framework.Framework, testParams benchmarkTestParams) {
	clusterAName := framework.TestContext.ClusterIDs[testParams.ClientCluster]
	clusterBName := framework.TestContext.ClusterIDs[testParams.ServerCluster]
	var connectionTimeout uint = 5
	var connectionAttempts uint = 1
	var netperfPort = 12865

	framework.By(fmt.Sprintf("Creating a Nettest Server Pod on %q", clusterBName))
	nettestServerPod := f.NewNetworkPod(&framework.NetworkPodConfig{
		Type:               framework.LatencyServerPod,
		Cluster:            testParams.ServerCluster,
		Scheduling:         testParams.ServerPodScheduling,
		ConnectionTimeout:  connectionTimeout,
		ConnectionAttempts: connectionAttempts,
		Port:               netperfPort,
	})

	podsClusterB := framework.KubeClients[testParams.ServerCluster].CoreV1().Pods(f.Namespace)
	p1, _ := podsClusterB.Get(nettestServerPod.Pod.Name, metav1.GetOptions{})
	By(fmt.Sprintf("Nettest Server Pod %q was created on node %q", nettestServerPod.Pod.Name, nettestServerPod.Pod.Spec.NodeName))

	remoteIP := p1.Status.PodIP
	if framework.TestContext.GlobalnetEnabled {
		By(fmt.Sprintf("Pointing a service ClusterIP to the listener pod in cluster %q",
			framework.TestContext.ClusterIDs[testParams.ServerCluster]))
		service := nettestServerPod.CreateService()

		// Wait for the globalIP annotation on the service.
		service = f.AwaitUntilAnnotationOnService(testParams.ServerCluster, globalnetGlobalIPAnnotation, service.Name, service.Namespace)
		remoteIP = service.GetAnnotations()[globalnetGlobalIPAnnotation]

		By(fmt.Sprintf("remote service IP is %q, service name %q, %v", remoteIP, service.Name, service))
	}

	nettestClientPod := f.NewNetworkPod(&framework.NetworkPodConfig{
		Type:               framework.LatencyClientPod,
		Cluster:            testParams.ClientCluster,
		Scheduling:         testParams.ClientPodScheduling,
		RemoteIP:           remoteIP,
		ConnectionTimeout:  connectionTimeout,
		ConnectionAttempts: connectionAttempts,
		Port:               netperfPort,
	})

	if framework.TestContext.GlobalnetEnabled {
		// Wait for the globalIP annotation on the client pod
		nettestClientPod.Pod = f.AwaitUntilAnnotationOnPod(testParams.ClientCluster, globalnetGlobalIPAnnotation,
			nettestClientPod.Pod.Name, nettestClientPod.Pod.Namespace)
	}

	framework.By(fmt.Sprintf("Nettest Client Pod %q was created on cluster %q, node %q; connect to server pod ip %q",
		nettestClientPod.Pod.Name, clusterAName, nettestClientPod.Pod.Spec.NodeName, remoteIP))

	framework.By(fmt.Sprintf("Waiting for the client pod %q to exit, returning what client sent", nettestClientPod.Pod.Name))
	nettestClientPod.AwaitFinishVerbose(Verbose)
}
