/*
© 2021 Red Hat, Inc. and others.

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
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	TCPSniffMetricsCommand = "tcpdump -ln -c 5 -i any tcp and src port 9898 and dst port 8080 and 'tcp[tcpflags] == tcp-syn'"
)

var validateFirewallMetricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Validate firewall access to metrics.",
	Long:  "This command checks whether firewall configuration allows metrics to be read from the Gateway nodes.",
	Run:   validateFirewallMetricsConfig,
}

func init() {
	addValidateFWConfigFlags(validateFirewallMetricsCmd)
	validateFirewallMetricsCmd.Flags().BoolVar(&verboseOutput, "verbose", false,
		"produce verbose logs during validation")
	validateFirewallConfigCmd.AddCommand(validateFirewallMetricsCmd)
}

func validateFirewallMetricsConfig(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContexts)
	exitOnError("Error getting REST config for cluster", err)

	for _, item := range configs {
		validateFirewallMetricsConfigWithinCluster(item.config, item.clusterName)
	}
}

func validateFirewallMetricsConfigWithinCluster(config *rest.Config, clusterName string) {
	status.Start(fmt.Sprintf("Validating the firewall configuration to check if metrics port (8080)"+
		" is allowed in cluster %q", clusterName))

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		message := fmt.Sprintf("Error creating API server client: %s", err)
		status.QueueFailureMessage(message)
		return
	}

	podCommand := fmt.Sprintf("timeout %d %s", validationTimeout, TCPSniffMetricsCommand)
	sPod, err := spawnSnifferPodOnGatewayNode(clientSet, namespace, podCommand)
	if err != nil {
		message := fmt.Sprintf("Error while spawning the sniffer pod on the GatewayNode: %v", err)
		status.QueueFailureMessage(message)
		return
	}

	defer sPod.DeletePod()
	gatewayPodIP := sPod.Pod.Status.HostIP
	podCommand = fmt.Sprintf("for i in $(seq 10); do timeout 2 nc -p 9898 %s 8080; done", gatewayPodIP)
	cPod, err := spawnClientPodOnNonGatewayNode(clientSet, namespace, podCommand)
	if err != nil {
		message := fmt.Sprintf("Error while spawning the client pod on non-Gateway node: %v", err)
		status.QueueFailureMessage(message)
		return
	}

	defer cPod.DeletePod()
	if err = cPod.AwaitPodCompletion(); err != nil {
		message := fmt.Sprintf("Error while waiting for client pod to be finish its execution: %v", err)
		status.QueueFailureMessage(message)
		return
	}

	if err = sPod.AwaitPodCompletion(); err != nil {
		message := fmt.Sprintf("Error while waiting for sniffer pod to be finish its execution: %v", err)
		status.QueueFailureMessage(message)
		return
	}

	if verboseOutput {
		status.QueueSuccessMessage("tcpdump output from Sniffer Pod on Gateway node")
		status.QueueSuccessMessage(sPod.PodOutput)
	}

	// Verify that tcpdump output (i.e, from snifferPod) contains the HostIP of clientPod
	if !strings.Contains(sPod.PodOutput, cPod.Pod.Status.HostIP) {
		message := fmt.Sprintf("The tcpdump output from the sniffer pod does not contain the"+
			" client pod HostIP. Please check that your firewall configuration allows TCP/8080 traffic"+
			" on the %q node.", sPod.Pod.Spec.NodeName)
		status.QueueFailureMessage(message)
		return
	}

	status.QueueSuccessMessage("Prometheus metrics can be retrieved from gateway nodes.")
	status.End(status.ResultFromMessages())
}
