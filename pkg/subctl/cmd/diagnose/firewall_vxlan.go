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

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"

	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
)

const (
	TCPSniffVxLANCommand = "tcpdump -ln -c 3 -i vx-submariner tcp and port 8080 and 'tcp[tcpflags] == tcp-syn'"
)

func init() {
	command := &cobra.Command{
		Use:   "intra-cluster",
		Short: "Check firewall access for intra-cluster Submariner VxLAN traffic",
		Long:  "This command checks if the firewall configuration allows traffic over vx-submariner interface.",
		Run: func(command *cobra.Command, args []string) {
			cmd.ExecuteMultiCluster(checkVxLANConfig)
		},
	}

	addDiagnoseFWConfigFlags(command)
	addVerboseFlag(command)
	diagnoseFirewallConfigCmd.AddCommand(command)
}

func checkVxLANConfig(cluster *cmd.Cluster) bool {
	status := cli.NewStatus()

	if cluster.Submariner == nil {
		status.Start(cmd.SubmMissingMessage)
		status.End(cli.Warning)
		return true
	}

	status.Start("Checking the firewall configuration to determine if VXLAN traffic is allowed")

	if isClusterSingleNode(cluster, status) {
		// Skip the check if it's a single node cluster
		return true
	}

	checkFWConfig(cluster, status)

	if status.HasFailureMessages() {
		status.End(cli.Failure)
		return false
	}

	status.EndWithSuccess("The firewall configuration allows VXLAN traffic")

	return true
}

func checkFWConfig(cluster *cmd.Cluster, status *cli.Status) {
	if cluster.Submariner.Status.NetworkPlugin == "OVNKubernetes" {
		status.QueueSuccessMessage("This check is not necessary for the OVNKubernetes CNI plugin")
		return
	}

	localEndpoint := getLocalEndpointResource(cluster, status)
	if localEndpoint == nil {
		return
	}

	remoteEndpoint := getAnyRemoteEndpointResource(cluster, status)
	if remoteEndpoint == nil {
		return
	}

	gwNodeName := getActiveGatewayNodeName(cluster, localEndpoint.Spec.Hostname, status)
	if gwNodeName == "" {
		return
	}

	podCommand := fmt.Sprintf("timeout %d %s", validationTimeout, TCPSniffVxLANCommand)
	sPod, err := spawnSnifferPodOnNode(cluster.KubeClient, gwNodeName, podNamespace, podCommand)
	if err != nil {
		status.EndWithFailure("Error spawning the sniffer pod on the Gateway node: %v", err)
		return
	}

	defer sPod.DeletePod()

	remoteClusterIP := strings.Split(remoteEndpoint.Spec.Subnets[0], "/")[0]
	podCommand = fmt.Sprintf("nc -w %d %s 8080", validationTimeout/2, remoteClusterIP)
	cPod, err := spawnClientPodOnNonGatewayNode(cluster.KubeClient, podNamespace, podCommand)
	if err != nil {
		status.QueueFailureMessage(fmt.Sprintf("Error spawning the client pod on non-Gateway node: %v", err))
		return
	}

	defer cPod.DeletePod()

	if err = cPod.AwaitPodCompletion(); err != nil {
		status.QueueFailureMessage(fmt.Sprintf("Error waiting for the client pod to finish its execution: %v", err))
		return
	}

	if err = sPod.AwaitPodCompletion(); err != nil {
		status.QueueFailureMessage(fmt.Sprintf("Error waiting for the sniffer pod to finish its execution: %v", err))
		return
	}

	if verboseOutput {
		status.QueueSuccessMessage("tcpdump output from the sniffer pod on Gateway node")
		status.QueueSuccessMessage(sPod.PodOutput)
	}

	// Verify that tcpdump output (i.e, from snifferPod) contains the remoteClusterIP
	if !strings.Contains(sPod.PodOutput, remoteClusterIP) {
		status.QueueFailureMessage(fmt.Sprintf("The tcpdump output from the sniffer pod does not contain the expected remote"+
			" endpoint IP %s. Please check that your firewall configuration allows UDP/4800 traffic.", remoteClusterIP))
		return
	}

	// Verify that tcpdump output (i.e, from snifferPod) contains the clientPod IPaddress
	if !strings.Contains(sPod.PodOutput, cPod.Pod.Status.PodIP) {
		status.QueueFailureMessage(fmt.Sprintf("The tcpdump output from the sniffer pod does not contain the client pod's IP."+
			" There seems to be some issue with the IPTable rules programmed on the %q node", cPod.Pod.Spec.NodeName))
		return
	}
}
