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

	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
)

const (
	tcpSniffVxLANCommand = "tcpdump -ln -c 3 -i vx-submariner tcp and port 8080 and 'tcp[tcpflags] == tcp-syn'"
)

func VxLANConfig(clusterInfo *cluster.Info, options FirewallOptions, status reporter.Interface) bool {
	mustHaveSubmariner(clusterInfo)

	status.Start("Checking the firewall configuration to determine if VXLAN traffic is allowed")
	defer status.End()

	singleNode, err := clusterInfo.HasSingleNode()
	if err != nil {
		status.Failure(err.Error())
		return false
	}

	if singleNode {
		status.Success(singleNodeMessage)
		return true
	}

	tracker := reporter.NewTracker(status)

	checkFWConfig(clusterInfo, options, tracker)

	if tracker.HasFailures() {
		return false
	}

	status.Success("The firewall configuration allows VXLAN traffic")

	return true
}

func checkFWConfig(clusterInfo *cluster.Info, options FirewallOptions, status reporter.Interface) {
	if clusterInfo.Submariner.Status.NetworkPlugin == "OVNKubernetes" {
		status.Success("This check is not necessary for the OVNKubernetes CNI plugin")
		return
	}

	localEndpoint, err := clusterInfo.GetLocalEndpoint()
	if err != nil {
		status.Failure("Unable to obtain the local endpoint: %v", err)
		return
	}

	remoteEndpoint, err := clusterInfo.GetAnyRemoteEndpoint()
	if err != nil {
		status.Failure("Unable to obtain a remote endpoint: %v", err)
		return
	}

	gwNodeName := getActiveGatewayNodeName(clusterInfo, localEndpoint.Spec.Hostname, status)
	if gwNodeName == "" {
		return
	}

	podCommand := fmt.Sprintf("timeout %d %s", options.ValidationTimeout, tcpSniffVxLANCommand)

	sPod, err := spawnSnifferPodOnNode(clusterInfo.ClientProducer.ForKubernetes(), gwNodeName, options.PodNamespace, podCommand)
	if err != nil {
		status.Failure("Error spawning the sniffer pod on the Gateway node: %v", err)
		return
	}

	defer sPod.Delete()

	remoteClusterIP := strings.Split(remoteEndpoint.Spec.Subnets[0], "/")[0]
	podCommand = fmt.Sprintf("nc -w %d %s 8080", options.ValidationTimeout/2, remoteClusterIP)

	cPod, err := spawnClientPodOnNonGatewayNode(clusterInfo.ClientProducer.ForKubernetes(), options.PodNamespace, podCommand)
	if err != nil {
		status.Failure("Error spawning the client pod on non-Gateway node: %v", err)
		return
	}

	defer cPod.Delete()

	if err = cPod.AwaitCompletion(); err != nil {
		status.Failure("Error waiting for the client pod to finish its execution: %v", err)
		return
	}

	if err = sPod.AwaitCompletion(); err != nil {
		status.Failure("Error waiting for the sniffer pod to finish its execution: %v", err)
		return
	}

	if options.VerboseOutput {
		status.Success("tcpdump output from the sniffer pod on Gateway node")
		status.Success(sPod.PodOutput)
	}

	// Verify that tcpdump output (i.e, from snifferPod) contains the remoteClusterIP
	if !strings.Contains(sPod.PodOutput, remoteClusterIP) {
		status.Failure("The tcpdump output from the sniffer pod does not contain the expected remote"+
			" endpoint IP %s. Please check that your firewall configuration allows UDP/4800 traffic.", remoteClusterIP)
		return
	}

	// Verify that tcpdump output (i.e, from snifferPod) contains the clientPod IPaddress
	if !strings.Contains(sPod.PodOutput, cPod.Pod.Status.PodIP) {
		status.Failure("The tcpdump output from the sniffer pod does not contain the client pod's IP."+
			" There seems to be some issue with the IPTable rules programmed on the %q node", cPod.Pod.Spec.NodeName)
	}
}
