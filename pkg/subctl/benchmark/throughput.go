package benchmark

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/submariner-io/shipyard/test/e2e/framework"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("[throughput] Test throughput between two Clusters", func() {
	f := framework.NewFramework("throughput")

	When("a pod sends throughput test to a pod in a remote cluster", func() {
		It("should be able to execute throughput test", func() {
			RunThroughputTest(f)
		})
	})
})

func RunThroughputTest(f *framework.Framework) {
	clusterAName := framework.TestContext.ClusterIDs[framework.ClusterA]
	clusterBName := framework.TestContext.ClusterIDs[framework.ClusterB]

	By(fmt.Sprintf("Creating a Nettest Server Pods on %q", clusterBName))
	netshootPodB := NewNettestSServerPod(f, framework.ClusterB, framework.GatewayNode)
	pods := framework.KubeClients[framework.ClusterB].CoreV1().Pods(f.Namespace)
	p1, _ := pods.Get(netshootPodB.Name, metav1.GetOptions{})

	remoteIP := p1.Status.PodIP

	By(fmt.Sprintf("Creating a Nettest Client Pods on %q , connecto to server pod ip %q", clusterAName, remoteIP))
	nettestPodA := NewNettestClientPod(f, framework.ClusterA, framework.GatewayNode, remoteIP)

	By(fmt.Sprintf("Waiting for the client pod %q to exit, returning what client sent", nettestPodA.Name))
	AwaitFinish(f, framework.ClusterA, nettestPodA)
}

func AwaitFinish(f *framework.Framework, cluster framework.ClusterIndex, pod *corev1.Pod) {
	pods := framework.KubeClients[cluster].CoreV1().Pods(f.Namespace)

	_, terminationErrorMsg, _ := framework.AwaitResultOrError(fmt.Sprintf("await pod %q finished", pod.Name), func() (interface{}, error) {
		return pods.Get(pod.Name, metav1.GetOptions{})
	}, func(result interface{}) (bool, string, error) {
		pod = result.(*corev1.Pod)

		switch pod.Status.Phase {
		case corev1.PodSucceeded:
			return true, "", nil
		case corev1.PodFailed:
			return true, "", nil
		default:
			return false, fmt.Sprintf("Pod status is %v", pod.Status.Phase), nil
		}
	})

	finished := pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed
	if finished {
		terminationMessage := pod.Status.ContainerStatuses[0].State.Terminated.Message

		framework.Logf("Pod %q output:\n%s", pod.Name, terminationMessage)
		if terminationErrorMsg != "" {
			framework.Logf("Pod %q error:\n%s", pod.Name, terminationErrorMsg)
		}
	}
}

func AwaitReady(f *framework.Framework, cluster framework.ClusterIndex, pod *corev1.Pod) {
	pods := framework.KubeClients[cluster].CoreV1().Pods(f.Namespace)

	framework.AwaitUntil("await pod ready", func() (interface{}, error) {
		return pods.Get(pod.Name, metav1.GetOptions{})
	}, func(result interface{}) (bool, string, error) {
		p1 := result.(*corev1.Pod)
		if p1.Status.Phase != corev1.PodRunning {
			if p1.Status.Phase != corev1.PodPending {
				return false, "", fmt.Errorf("unexpected pod phase %v - expected %v or %v", pod.Status.Phase, corev1.PodPending, corev1.PodRunning)
			}
			return false, fmt.Sprintf("Pod %q is still pending", p1.Name), nil
		}
		return true, "", nil // pod is running
	})
}

func NewNettestSServerPod(f *framework.Framework, cluster framework.ClusterIndex, scheduling framework.NetworkPodScheduling) *corev1.Pod {
	nettestPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "nettest",
			Labels: map[string]string{
				"run": "nettest",
			},
		},
		Spec: corev1.PodSpec{
			Affinity:      nodeAffinity(scheduling),
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:            "nettest",
					Image:           "quay.io/submariner/nettest:devel",
					ImagePullPolicy: corev1.PullAlways,
					Command:         []string{"iperf3", "-s"},
				},
			},
		},
	}
	pc := framework.KubeClients[cluster].CoreV1().Pods(f.Namespace)
	var err error
	pod, err := pc.Create(&nettestPod)
	Expect(err).NotTo(HaveOccurred())
	By(fmt.Sprintf("Waiting for the server pod %q to be ready", pod.Name))
	AwaitReady(f, cluster, pod)
	return pod
}

func NewNettestClientPod(f *framework.Framework, cluster framework.ClusterIndex, scheduling framework.NetworkPodScheduling,
	targetIp string) *corev1.Pod {
	nettestPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "nettest-client-pod",
			Labels: map[string]string{
				"run": "nettest-client-pod",
			},
		},
		Spec: corev1.PodSpec{
			Affinity:      nodeAffinity(scheduling),
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:            "nettest-client-pod",
					Image:           "quay.io/submariner/nettest:devel",
					ImagePullPolicy: corev1.PullAlways,
					Command: []string{
						"sh", "-c", "echo [throughput] client says test | iperf3 -w 256K -P 10 -c $TARGET_IP >/dev/termination-log 2>&1"},
					Env: []corev1.EnvVar{
						{Name: "TARGET_IP", Value: targetIp},
					},
				},
			},
		},
	}
	pc := framework.KubeClients[cluster].CoreV1().Pods(f.Namespace)
	var err error
	pod, err := pc.Create(&nettestPod)
	Expect(err).NotTo(HaveOccurred())
	return pod
}

func nodeAffinity(scheduling framework.NetworkPodScheduling) *corev1.Affinity {
	var nodeSelTerms []corev1.NodeSelectorTerm

	switch scheduling {
	case framework.GatewayNode:
		nodeSelTerms = addNodeSelectorTerm(nodeSelTerms, framework.GatewayLabel, corev1.NodeSelectorOpIn, []string{"true"})

	case framework.NonGatewayNode:
		nodeSelTerms = addNodeSelectorTerm(nodeSelTerms, framework.GatewayLabel, corev1.NodeSelectorOpDoesNotExist, nil)
		nodeSelTerms = addNodeSelectorTerm(nodeSelTerms, framework.GatewayLabel, corev1.NodeSelectorOpNotIn, []string{"true"})
	}

	return &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: nodeSelTerms,
			},
		},
	}
}

func addNodeSelectorTerm(nodeSelTerms []corev1.NodeSelectorTerm, label string,
	op corev1.NodeSelectorOperator, values []string) []corev1.NodeSelectorTerm {
	return append(nodeSelTerms, corev1.NodeSelectorTerm{MatchExpressions: []corev1.NodeSelectorRequirement{
		{
			Key:      label,
			Operator: op,
			Values:   values,
		},
	}})
}
