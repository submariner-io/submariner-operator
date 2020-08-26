package benchmark

import (
	"fmt"

	"github.com/onsi/gomega"
	"github.com/submariner-io/shipyard/test/e2e/framework"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func StartThroughputTests() {
	gomega.RegisterFailHandler(func(message string, callerSkip ... int) { panic(message) })
	f := framework.NewBareFramework("throughput")
	framework.ValidateFlags(framework.TestContext)
	framework.BeforeSuite()

	f.BeforeEach()
	framework.By("Performing throughput tests from Gateway pod to Gateway pod")
	RunThroughputTest(f,  framework.GatewayNode)
	framework.By("Performing throughput tests from Non-Gateway pod to Non-Gateway pod")
	RunThroughputTest(f,  framework.NonGatewayNode)
	f.AfterEach()
	framework.RunCleanupActions()
}

func RunThroughputTest(f *framework.Framework, scheduling framework.NetworkPodScheduling) {
	clusterAName := framework.TestContext.ClusterIDs[framework.ClusterA]
	clusterBName := framework.TestContext.ClusterIDs[framework.ClusterB]
	var connectionTimeout uint = 5
	var connectionAttempts uint = 1

	framework.By(fmt.Sprintf("Creating a Nettest Server Pod on %q", clusterBName))
	nettestPodB := f.NewNetworkPod(&framework.NetworkPodConfig{
		Type: framework.ThroughputServerPod,
		Cluster: framework.ClusterB,
		Scheduling: scheduling,
		ConnectionTimeout: connectionTimeout,
		ConnectionAttempts: connectionAttempts,
	})

	podsClusterB := framework.KubeClients[framework.ClusterB].CoreV1().Pods(f.Namespace)
	p1, _ := podsClusterB.Get(nettestPodB.Pod.Name, metav1.GetOptions{})
	framework.By(fmt.Sprintf("Nettest Server Pod %q was created on node %q", nettestPodB.Pod.Name, nettestPodB.Pod.Spec.NodeName))

	remoteIP := p1.Status.PodIP

	framework.By(fmt.Sprintf("Creating a Nettest Client Pod on %q , connect to server pod ip %q", clusterAName, remoteIP))
	nettestPodA := f.NewNetworkPod(&framework.NetworkPodConfig{
		Type: framework.ThroughputClientPod,
		Cluster: framework.ClusterA,
		Scheduling: scheduling,
		RemoteIP: remoteIP,
		ConnectionTimeout: connectionTimeout,
		ConnectionAttempts: connectionAttempts,
	})

	framework.By(fmt.Sprintf("Nettest Client Pod %q was created on node %q", nettestPodA.Pod.Name, nettestPodA.Pod.Spec.NodeName))

	framework.By(fmt.Sprintf("Waiting for the client pod %q to exit, returning what client sent", nettestPodA.Pod.Name))
	nettestPodA.AwaitFinish()
}
