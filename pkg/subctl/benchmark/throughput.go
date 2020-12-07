package benchmark

import (
	"fmt"

	"github.com/onsi/gomega"
	"github.com/submariner-io/shipyard/test/e2e/framework"

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
	var connectionTimeout uint = 20
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

	if framework.TestContext.GlobalnetEnabled && testParams.ClientCluster != testParams.ServerCluster {
		By(fmt.Sprintf("Pointing a service ClusterIP to the listener pod in cluster %q",
			framework.TestContext.ClusterIDs[testParams.ServerCluster]))
		service := nettestServerPod.CreateService()

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
}
