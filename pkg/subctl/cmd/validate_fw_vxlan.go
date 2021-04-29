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
	"os"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
)

var validateFirewallVxLANConfigCmd = &cobra.Command{
	Use:   "vxlan",
	Short: "Check firewall access for Submariner VXLAN traffic",
	Long:  "This command checks if the firewall configuration allows traffic via the Submariner VXLAN interface.",
	Run:   validateFirewallVxLANConfig,
}

const (
	TCPSniffVxLANCommand = "tcpdump -ln -c 3 -i vx-submariner tcp and port 8080 and 'tcp[tcpflags] == tcp-syn'"
)

func init() {
	addValidateFWConfigFlags(validateFirewallVxLANConfigCmd)
	validateFirewallVxLANConfigCmd.Flags().BoolVar(&verboseOutput, "verbose", false,
		"produce verbose logs during validation")
	validateFirewallConfigCmd.AddCommand(validateFirewallVxLANConfigCmd)
}

func validateFirewallVxLANConfig(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContexts)
	exitOnError("Error getting REST config for cluster", err)

	validationStatus := true

	for _, item := range configs {
		status.Start(fmt.Sprintf("Retrieving Submariner resource from %q", item.clusterName))
		submariner := getSubmarinerResource(item.config)
		if submariner == nil {
			status.QueueWarningMessage(submMissingMessage)
			status.End(cli.Warning)
			continue
		}

		status.End(cli.Success)
		validationStatus = validationStatus && validateVxLANConfigWithinCluster(item.config, item.clusterName, submariner)
	}

	if !validationStatus {
		os.Exit(1)
	}
}

func validateVxLANConfigWithinCluster(config *rest.Config, clusterName string, submariner *v1alpha1.Submariner) bool {
	status.Start(fmt.Sprintf("Checking the firewall configuration to determine if VXLAN traffic is allowed"+
		" in cluster %q", clusterName))
	validationStatus := validateFWConfigWithinCluster(config, submariner)
	status.End(status.ResultFromMessages())
	return validationStatus
}

func validateFWConfigWithinCluster(config *rest.Config, submariner *v1alpha1.Submariner) bool {
	if submariner.Status.NetworkPlugin == "OVNKubernetes" {
		status.QueueSuccessMessage("This check is not necessary for the OVNKubernetes CNI plugin")
		return true
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		message := fmt.Sprintf("Error creating API server client: %s", err)
		status.QueueFailureMessage(message)
		return false
	}

	gateways := getGatewaysResource(config)
	if gateways == nil || len(gateways.Items) == 0 {
		status.QueueWarningMessage("There are no gateways detected on the cluster.")
		return false
	}

	if len(gateways.Items[0].Status.Connections) == 0 {
		status.QueueWarningMessage("There are no active connections to remote clusters.")
		return false
	}

	podCommand := fmt.Sprintf("timeout %d %s", validationTimeout, TCPSniffVxLANCommand)
	sPod, err := spawnSnifferPodOnGatewayNode(clientSet, namespace, podCommand)
	if err != nil {
		message := fmt.Sprintf("Error while spawning the sniffer pod on the GatewayNode: %v", err)
		status.QueueFailureMessage(message)
		return false
	}

	defer sPod.DeletePod()
	remoteClusterIP := strings.Split(gateways.Items[0].Status.Connections[0].Endpoint.Subnets[0], "/")[0]
	podCommand = fmt.Sprintf("nc -w %d %s 8080", validationTimeout/2, remoteClusterIP)
	cPod, err := spawnClientPodOnNonGatewayNode(clientSet, namespace, podCommand)
	if err != nil {
		message := fmt.Sprintf("Error while spawning the client pod on non-Gateway node: %v", err)
		status.QueueFailureMessage(message)
		return false
	}

	defer cPod.DeletePod()
	if err = cPod.AwaitPodCompletion(); err != nil {
		message := fmt.Sprintf("Error while waiting for client pod to be finish its execution: %v", err)
		status.QueueFailureMessage(message)
		return false
	}

	if err = sPod.AwaitPodCompletion(); err != nil {
		message := fmt.Sprintf("Error while waiting for sniffer pod to be finish its execution: %v", err)
		status.QueueFailureMessage(message)
		return false
	}

	if verboseOutput {
		status.QueueSuccessMessage("tcpdump output from Sniffer Pod on Gateway node")
		status.QueueSuccessMessage(sPod.PodOutput)
	}

	// Verify that tcpdump output (i.e, from snifferPod) contains the remoteClusterIP
	if !strings.Contains(sPod.PodOutput, remoteClusterIP) {
		message := fmt.Sprintf("The tcpdump output from the sniffer pod does not contain the expected remote"+
			" endpoint IP %s. Please check that your firewall configuration allows UDP/4800 traffic.", remoteClusterIP)
		status.QueueFailureMessage(message)
		return false
	}

	// Verify that tcpdump output (i.e, from snifferPod) contains the clientPod IPaddress
	if !strings.Contains(sPod.PodOutput, cPod.Pod.Status.PodIP) {
		message := fmt.Sprintf("The tcpdump output from the sniffer pod does not contain the client pod's IP."+
			" There seems to be some issue with the IPTable rules programmed on the %q node", cPod.Pod.Spec.NodeName)
		status.QueueFailureMessage(message)
		return false
	}

	status.QueueSuccessMessage("The firewall configuration for vx-submariner is working fine.")
	return true
}
