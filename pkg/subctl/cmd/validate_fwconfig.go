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
	"github.com/submariner-io/shipyard/test/e2e/framework"
	"k8s.io/client-go/kubernetes"

	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/resource"
)

var validateIntraClusterFWConfig = &cobra.Command{
	Use:   "fw-intra-cluster",
	Short: "Validate the Firewall Configuration within the cluster.",
	Long:  "This command checks whether firewall configuration allows traffic via vx-submariner interface.",
	Run:   validateFWConfig,
}

var validationTimeout uint

const (
	TCPSniffCommand = "tcpdump -ln -c 3 -i vx-submariner tcp and port 8080 and 'tcp[tcpflags] == tcp-syn'"
)

func addValidateFWConfigFlags(cmd *cobra.Command) {
	cmd.Flags().UintVar(&validationTimeout, "validation-timeout", 90,
		"timeout in seconds while validating the connection attempt")
}

func init() {
	addValidateFWConfigFlags(validateIntraClusterFWConfig)
	validateCmd.AddCommand(validateIntraClusterFWConfig)
}

func validateFWConfig(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContext)
	exitOnError("Error getting REST config for cluster", err)

	for _, item := range configs {
		status.Start(fmt.Sprintf("Retrieving Submariner resource from %q", item.clusterName))
		submariner := getSubmarinerResource(item.config)
		if submariner == nil {
			status.QueueWarningMessage(submMissingMessage)
			status.End(cli.Success)
			continue
		}

		status.End(cli.Success)
		retVal := validateFWConfigWithinCluster(item, submariner)
		if retVal.HasFailureMessages() {
			status.End(cli.Failure)
			continue
		} else {
			status.End(cli.Success)
		}
	}
}

func validateFWConfigWithinCluster(item restConfig, submariner *v1alpha1.Submariner) *cli.Status {
	status.Start(fmt.Sprintf("Validating Firewall configuration in cluster %q", item.clusterName))
	if submariner.Status.NetworkPlugin == "OVNKubernetes" {
		status.QueueSuccessMessage("This validation is not necessary for OVNKubernetes.")
		return status
	}

	clientSet, err := kubernetes.NewForConfig(item.config)
	if err != nil {
		message := fmt.Sprintf("Error creating API server client: %s", err)
		status.QueueFailureMessage(message)
		return status
	}

	gateways := getGatewaysResource(item.config)
	if gateways == nil {
		status.QueueWarningMessage("There are no gateways detected on the cluster.")
		return status
	}

	if len(gateways.Items[0].Status.Connections) == 0 {
		status.QueueWarningMessage("There are no active connections to remote clusters.")
		return status
	}

	sPod, err := spawnSnifferPodOnGatewayNode(clientSet)
	if err != nil {
		message := fmt.Sprintf("Error while spawning the Sniffer Pod on the GatewayNode. %v", err)
		status.QueueFailureMessage(message)
		return status
	}

	defer sPod.DeletePod()
	remoteClusterIP := strings.Split(gateways.Items[0].Status.Connections[0].Endpoint.Subnets[0], "/")[0]
	cPod, err := spawnClientPodOnNonGatewayNode(clientSet, remoteClusterIP)
	if err != nil {
		message := fmt.Sprintf("Error while spawning the Client Pod on nonGateway node. %v", err)
		status.QueueFailureMessage(message)
		return status
	}

	defer cPod.DeletePod()
	if err = cPod.AwaitUntilPodCompletion(); err != nil {
		message := fmt.Sprintf("Error while waiting for Client Pod to be finish its execution. %v", err)
		status.QueueFailureMessage(message)
		return status
	}

	if err = sPod.AwaitUntilPodCompletion(); err != nil {
		message := fmt.Sprintf("Error while waiting for Sniffer Pod to be finish its execution. %v", err)
		status.QueueFailureMessage(message)
		return status
	}

	// Verify that tcpdump output (i.e, from snifferPod) contains the remoteClusterIP
	if !strings.Contains(sPod.PodOutput, remoteClusterIP) {
		message := fmt.Sprintf("Tcpdump output from Sniffer Pod does not include the expected remoteClusterIP." +
			" Please check your Firewall configuration to allow UDP/4800 traffic.")
		status.QueueFailureMessage(message)
		return status
	}

	// Verify that tcpdump output (i.e, from snifferPod) contains the clientPod IPaddress
	if !strings.Contains(sPod.PodOutput, cPod.Pod.Status.PodIP) {
		message := fmt.Sprintf("Tcpdump output from Sniffer Pod does not include the clientPod IPAddress."+
			" There seems to be some issue with the IPTable rules programmed on the %q node", cPod.Pod.Spec.NodeName)
		status.QueueFailureMessage(message)
		return status
	}

	status.QueueSuccessMessage("Firewall configuration within the cluster looks fine.")
	return status
}

func spawnSnifferPodOnGatewayNode(clientSet *kubernetes.Clientset) (*resource.NetworkPod, error) {
	podCommand := fmt.Sprintf("timeout %d %s", validationTimeout, TCPSniffCommand)
	sPod, err := resource.SchedulePod(&resource.PodConfig{
		Name:       "validate-fwconfig-sniffer",
		ClientSet:  clientSet,
		Scheduling: framework.GatewayNode,
		Networking: framework.HostNetworking,
		Namespace:  namespace,
		Command:    podCommand,
	})

	if err != nil {
		return nil, err
	}
	return sPod, nil
}

func spawnClientPodOnNonGatewayNode(clientSet *kubernetes.Clientset, remoteIP string) (*resource.NetworkPod, error) {
	podCommand := fmt.Sprintf("nc -w %d %s 8080", validationTimeout/2, remoteIP)
	cPod, err := resource.SchedulePod(&resource.PodConfig{
		Name:       "validate-fwconfig-client",
		ClientSet:  clientSet,
		Scheduling: framework.NonGatewayNode,
		Networking: framework.PodNetworking,
		Namespace:  namespace,
		Command:    podCommand,
	})

	if err != nil {
		return nil, err
	}
	return cPod, nil
}
