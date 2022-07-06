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
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
)

const (
	TCPSniffMetricsCommand = "tcpdump -ln -c 5 -i any tcp and src port 9898 and dst port 8080 and 'tcp[tcpflags] == tcp-syn'"
)

func init() {
	command := &cobra.Command{
		Use:   "metrics",
		Short: "Check firewall access to metrics",
		Long:  "This command checks if the firewall configuration allows metrics to be accessed from the Gateway nodes.",
		Run: func(command *cobra.Command, args []string) {
			cmd.ExecuteMultiCluster(restConfigProducer, checkFirewallMetricsConfig)
		},
	}

	addDiagnoseFWConfigFlags(command)
	addVerboseFlag(command)

	diagnoseFirewallConfigCmd.AddCommand(command)
}

func checkFirewallMetricsConfig(cluster *cmd.Cluster) bool {
	status := cli.NewStatus()

	status.Start("Checking the firewall configuration to determine if the metrics port (8080) is allowed")

	if isClusterSingleNode(cluster, status) {
		// Skip the check if it's a single node cluster
		return true
	}

	podCommand := fmt.Sprintf("timeout %d %s", validationTimeout, TCPSniffMetricsCommand)

	sPod, err := spawnSnifferPodOnGatewayNode(cluster.KubeClient, podNamespace, podCommand)
	if err != nil {
		status.EndWithFailure("Error spawning the sniffer pod on the Gateway node: %v", err)
		return false
	}

	defer sPod.Delete()

	gatewayPodIP := sPod.Pod.Status.HostIP
	podCommand = fmt.Sprintf("for i in $(seq 10); do timeout 2 nc -p 9898 %s 8080; done", gatewayPodIP)

	cPod, err := spawnClientPodOnNonGWNodeWithHostNwk(cluster.KubeClient, podNamespace, podCommand)
	if err != nil {
		status.EndWithFailure("Error spawning the client pod on non-Gateway node: %v", err)
		return false
	}

	defer cPod.Delete()

	if err = cPod.AwaitCompletion(); err != nil {
		status.EndWithFailure("Error waiting for the client pod to finish its execution: %v", err)
		return false
	}

	if err = sPod.AwaitCompletion(); err != nil {
		status.EndWithFailure("Error waiting for the sniffer pod to finish its execution: %v", err)
		return false
	}

	if verboseOutput {
		status.QueueSuccessMessage("tcpdump output from sniffer pod on Gateway node")
		status.QueueSuccessMessage(sPod.PodOutput)
	}

	// Verify that tcpdump output (i.e, from snifferPod) contains the HostIP of clientPod
	if !strings.Contains(sPod.PodOutput, cPod.Pod.Status.HostIP) {
		status.EndWithFailure("The tcpdump output from the sniffer pod does not contain the"+
			" client pod HostIP. Please check that your firewall configuration allows TCP/8080 traffic"+
			" on the %q node.", sPod.Pod.Spec.NodeName)

		return false
	}

	if status.HasFailureMessages() {
		status.EndWith(cli.Failure)
		return false
	}

	status.EndWithSuccess("The firewall configuration allows metrics to be retrieved from Gateway nodes")

	return true
}
