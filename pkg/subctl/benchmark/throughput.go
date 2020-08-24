package benchmark

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	"github.com/submariner-io/shipyard/test/e2e/framework"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("[throughput] Test throughput between two Clusters", func() {
	f := framework.NewFramework("throughput")

	When("a gateway pod sends throughput test to a gateway pod in a remote cluster on a", func() {
		It("should be able to execute throughput test", func() {
			RunThroughputTest(f,  framework.GatewayNode)
		})
	})

	When("a non-gateway pod sends throughput test to a non-gateway pod in a remote cluster", func() {
		It("should be able to execute throughput test", func() {
			RunThroughputTest(f,  framework.NonGatewayNode)
		})
	})
})

func RunThroughputTest(f *framework.Framework, scheduling framework.NetworkPodScheduling) {
	clusterAName := framework.TestContext.ClusterIDs[framework.ClusterA]
	clusterBName := framework.TestContext.ClusterIDs[framework.ClusterB]
	var connectionTimeout uint = 5
	var connectionAttempts uint = 1

	By(fmt.Sprintf("Creating a Nettest Server Pod on %q", clusterBName))
	nettestPodB := f.NewNetworkPod(&framework.NetworkPodConfig{
		Type: framework.ThroughputServerPod,
		Cluster: framework.ClusterB,
		Scheduling: scheduling,
		ConnectionTimeout: connectionTimeout,
		ConnectionAttempts: connectionAttempts,
	})

	podsClusterB := framework.KubeClients[framework.ClusterB].CoreV1().Pods(f.Namespace)
	p1, _ := podsClusterB.Get(nettestPodB.Pod.Name, metav1.GetOptions{})
	By(fmt.Sprintf("Nettest Server Pod %q was created on node %q", nettestPodB.Pod.Name, p1.Spec.NodeName))

	remoteIP := p1.Status.PodIP

	By(fmt.Sprintf("Creating a Nettest Client Pod on %q , connecto to server pod ip %q", clusterAName, remoteIP))
	nettestPodA := f.NewNetworkPod(&framework.NetworkPodConfig{
		Type: framework.ThroughputClientPod,
		Cluster: framework.ClusterA,
		Scheduling: scheduling,
		RemoteIP: remoteIP,
		ConnectionTimeout: connectionTimeout,
		ConnectionAttempts: connectionAttempts,
	})

	By(fmt.Sprintf("Nettest Client Pod %q was created on node ", nettestPodA.Pod.Name))

	By(fmt.Sprintf("Waiting for the client pod %q to exit, returning what client sent", nettestPodA.Pod.Name))
	nettestPodA.AwaitFinish()
}
