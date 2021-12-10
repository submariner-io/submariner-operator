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
package benchmark

import (
	"context"
	"fmt"
	"strings"

	"github.com/onsi/gomega"
	"github.com/submariner-io/shipyard/test/e2e/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

	clusterAName := framework.TestContext.ClusterIDs[framework.ClusterA]

	if !intraCluster {
		clusterBName := framework.TestContext.ClusterIDs[framework.ClusterB]

		if framework.TestContext.GlobalnetEnabled {
			fmt.Println("Latency test is not supported with Globalnet enabled, skipping the test...")
			cleanupFramework(f)

			return
		}

		latencyTestParams := benchmarkTestParams{
			ClientCluster:       framework.ClusterA,
			ServerCluster:       framework.ClusterB,
			ServerPodScheduling: framework.GatewayNode,
			ClientPodScheduling: framework.GatewayNode,
		}

		fmt.Printf("Performing latency tests from Gateway pod on cluster %q to Gateway pod on cluster %q\n",
			clusterAName, clusterBName)
		runLatencyTest(f, latencyTestParams)

		latencyTestParams.ServerPodScheduling = framework.NonGatewayNode
		latencyTestParams.ClientPodScheduling = framework.NonGatewayNode

		fmt.Printf("Performing latency tests from Non-Gateway pod on cluster %q to Non-Gateway pod on cluster %q\n",
			clusterAName, clusterBName)
		runLatencyTest(f, latencyTestParams)
	} else {
		latencyTestIntraClusterParams := benchmarkTestParams{
			ClientCluster:       framework.ClusterA,
			ServerCluster:       framework.ClusterA,
			ServerPodScheduling: framework.GatewayNode,
			ClientPodScheduling: framework.NonGatewayNode,
		}
		fmt.Printf("Performing latency tests from Non-Gateway pod to Gateway pod on cluster %q\n", clusterAName)
		runLatencyTest(f, latencyTestIntraClusterParams)
	}

	cleanupFramework(f)
}

func runLatencyTest(f *framework.Framework, testParams benchmarkTestParams) {
	clusterAName := framework.TestContext.ClusterIDs[testParams.ClientCluster]
	clusterBName := framework.TestContext.ClusterIDs[testParams.ServerCluster]
	var connectionTimeout uint = 5
	var connectionAttempts uint = 1

	By(fmt.Sprintf("Creating a Nettest Server Pod on %q", clusterBName))

	nettestServerPod := f.NewNetworkPod(&framework.NetworkPodConfig{
		Type:               framework.LatencyServerPod,
		Cluster:            testParams.ServerCluster,
		Scheduling:         testParams.ServerPodScheduling,
		ConnectionTimeout:  connectionTimeout,
		ConnectionAttempts: connectionAttempts,
	})

	podsClusterB := framework.KubeClients[testParams.ServerCluster].CoreV1().Pods(f.Namespace)
	p1, _ := podsClusterB.Get(context.TODO(), nettestServerPod.Pod.Name, metav1.GetOptions{})
	By(fmt.Sprintf("Nettest Server Pod %q was created on node %q", nettestServerPod.Pod.Name, nettestServerPod.Pod.Spec.NodeName))

	remoteIP := p1.Status.PodIP

	nettestClientPod := f.NewNetworkPod(&framework.NetworkPodConfig{
		Type:               framework.LatencyClientPod,
		Cluster:            testParams.ClientCluster,
		Scheduling:         testParams.ClientPodScheduling,
		RemoteIP:           remoteIP,
		ConnectionTimeout:  connectionTimeout,
		ConnectionAttempts: connectionAttempts,
	})

	By(fmt.Sprintf("Nettest Client Pod %q was created on cluster %q, node %q; connect to server pod ip %q",
		nettestClientPod.Pod.Name, clusterAName, nettestClientPod.Pod.Spec.NodeName, remoteIP))

	By(fmt.Sprintf("Waiting for the client pod %q to exit, returning what client sent", nettestClientPod.Pod.Name))
	nettestClientPod.AwaitFinishVerbose(Verbose)
	nettestClientPod.CheckSuccessfulFinish()
	latencyHeaders := strings.Split(nettestClientPod.TerminationMessage, "\n")[1]
	headers := strings.Split(latencyHeaders, ",")
	latencyValues := strings.Split(nettestClientPod.TerminationMessage, "\n")[2]
	values := strings.Split(latencyValues, ",")

	for i, v := range headers {
		fmt.Printf("%s:\t%s\n", v, values[i])
	}
}
