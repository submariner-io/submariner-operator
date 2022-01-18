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

package diagnose

import (
	"fmt"
	"strings"

	"github.com/submariner-io/submariner-operator/pkg/cluster"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
)

const (
	TCPSniffMetricsCommand = "tcpdump -ln -c 5 -i any tcp and src port 9898 and dst port 8080 and 'tcp[tcpflags] == tcp-syn'"
)

func FirewallMetricsConfig(clusterInfo *cluster.Info, status reporter.Interface) bool {
	status.Start("Checking the firewall configuration to determine if the metrics port (8080) is allowed")
	defer status.End()

	if isClusterSingleNode(clusterInfo, status) {
		// Skip the check if it's a single node cluster
		return true
	}

	podCommand := fmt.Sprintf("timeout %d %s", ValidationTimeout, TCPSniffMetricsCommand)

	sPod, err := spawnSnifferPodOnGatewayNode(clusterInfo.ClientProducer.ForKubernetes(), KubeProxyPodNamespace, podCommand)
	if err != nil {
		status.Failure("Error spawning the sniffer pod on the Gateway node: %v", err)

		return false
	}

	defer sPod.Delete()

	gatewayPodIP := sPod.Pod.Status.HostIP
	podCommand = fmt.Sprintf("for i in $(seq 10); do timeout 2 nc -p 9898 %s 8080; done", gatewayPodIP)

	cPod, err := spawnClientPodOnNonGatewayNode(clusterInfo.ClientProducer.ForKubernetes(), KubeProxyPodNamespace, podCommand)
	if err != nil {
		status.Failure("Error spawning the client pod on non-Gateway node: %v", err)

		return false
	}

	defer cPod.Delete()

	if err = cPod.AwaitCompletion(); err != nil {
		status.Failure("Error waiting for the client pod to finish its execution: %v", err)

		return false
	}

	if err = sPod.AwaitCompletion(); err != nil {
		status.Failure("Error waiting for the sniffer pod to finish its execution: %v", err)

		return false
	}

	if VerboseOutput {
		status.Success("tcpdump output from sniffer pod on Gateway node")
		status.Success(sPod.PodOutput)
	}

	// Verify that tcpdump output (i.e, from snifferPod) contains the HostIP of clientPod
	if !strings.Contains(sPod.PodOutput, cPod.Pod.Status.HostIP) {
		status.Failure("The tcpdump output from the sniffer pod does not contain the"+
			" client pod HostIP. Please check that your firewall configuration allows TCP/8080 traffic"+
			" on the %q node.", sPod.Pod.Spec.NodeName)

		return false
	}

	status.Success("The firewall configuration allows metrics to be retrieved from Gateway nodes")

	return true
}
