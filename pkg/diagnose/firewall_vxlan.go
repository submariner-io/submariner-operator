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

	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
)

const (
	TCPSniffVxLANCommand = "tcpdump -ln -c 3 -i vx-submariner tcp and port 8080 and 'tcp[tcpflags] == tcp-syn'"
)

func VxLANConfig(clusterInfo *cluster.Info, status reporter.Interface) bool {
	if clusterInfo.Submariner == nil {
		status.Warning(constants.SubmMissingMessage)

		return true
	}

	status.Start("Checking the firewall configuration to determine if VXLAN traffic is allowed")
	defer status.End()

	if isClusterSingleNode(clusterInfo, status) {
		// Skip the check if it's a single node cluster
		return true
	}

	if FWConfigNotOK(clusterInfo, status) {
		return false
	}

	status.Success("The firewall configuration allows VXLAN traffic")

	return true
}

func FWConfigNotOK(clusterInfo *cluster.Info, status reporter.Interface) bool {
	if clusterInfo.Submariner.Status.NetworkPlugin == "OVNKubernetes" {
		status.Success("This check is not necessary for the OVNKubernetes CNI plugin")
		return false
	}

	localEndpoint, failed := getLocalEndpointResource(clusterInfo, status)
	if localEndpoint == nil || failed {
		return true
	}

	remoteEndpoint, failed := getAnyRemoteEndpointResource(clusterInfo, status)
	if remoteEndpoint == nil || failed {
		return true
	}

	gwNodeName, failed := getActiveGatewayNodeName(clusterInfo, localEndpoint.Spec.Hostname, status)
	if gwNodeName == "" || failed {
		return true
	}

	podCommand := fmt.Sprintf("timeout %d %s", ValidationTimeout, TCPSniffVxLANCommand)

	sPod, err := spawnSnifferPodOnNode(clusterInfo.ClientProducer.ForKubernetes(), gwNodeName, KubeProxyPodNamespace, podCommand)
	if err != nil {
		status.Failure("Error spawning the sniffer pod on the Gateway node: %v", err)
		return true
	}

	defer sPod.Delete()

	remoteClusterIP := strings.Split(remoteEndpoint.Spec.Subnets[0], "/")[0]
	podCommand = fmt.Sprintf("nc -w %d %s 8080", ValidationTimeout/2, remoteClusterIP)

	cPod, err := spawnClientPodOnNonGatewayNode(clusterInfo.ClientProducer.ForKubernetes(), KubeProxyPodNamespace, podCommand)
	if err != nil {
		status.Failure(fmt.Sprintf("Error spawning the client pod on non-Gateway node: %v", err))
		return true
	}

	defer cPod.Delete()

	if err = cPod.AwaitCompletion(); err != nil {
		status.Failure(fmt.Sprintf("Error waiting for the client pod to finish its execution: %v", err))
		return true
	}

	if err = sPod.AwaitCompletion(); err != nil {
		status.Failure(fmt.Sprintf("Error waiting for the sniffer pod to finish its execution: %v", err))
		return true
	}

	if VerboseOutput {
		status.Success("tcpdump output from the sniffer pod on Gateway node")
		status.Success(sPod.PodOutput)
	}

	// Verify that tcpdump output (i.e, from snifferPod) contains the remoteClusterIP
	if !strings.Contains(sPod.PodOutput, remoteClusterIP) {
		status.Failure(fmt.Sprintf("The tcpdump output from the sniffer pod does not contain the expected remote"+
			" endpoint IP %s. Please check that your firewall configuration allows UDP/4800 traffic.", remoteClusterIP))
		return true
	}

	// Verify that tcpdump output (i.e, from snifferPod) contains the clientPod IPaddress
	if !strings.Contains(sPod.PodOutput, cPod.Pod.Status.PodIP) {
		status.Failure(fmt.Sprintf("The tcpdump output from the sniffer pod does not contain the client pod's IP."+
			" There seems to be some issue with the IPTable rules programmed on the %q node", cPod.Pod.Spec.NodeName))

		return true
	}

	return false
}
