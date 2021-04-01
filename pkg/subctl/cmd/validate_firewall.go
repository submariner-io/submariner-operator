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
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	"github.com/submariner-io/submariner-operator/pkg/subctl/resource"
)

var validateFirewallConfigCmd = &cobra.Command{
	Use:   "firewall",
	Short: "Validate the firewall configuration in the cluster.",
	Long:  "This command checks whether the firewall is configured as per Submariner pre-requisites.",
}

var validationTimeout uint

func addValidateFWConfigFlags(cmd *cobra.Command) {
	cmd.Flags().UintVar(&validationTimeout, "validation-timeout", 90,
		"timeout in seconds while validating the connection attempt")
	cmd.Flags().StringVar(&namespace, "namespace", "default",
		"namespace in which validation pods should be deployed")
}

func init() {
	validateCmd.AddCommand(validateFirewallConfigCmd)
}

func spawnSnifferPodOnGatewayNode(clientSet *kubernetes.Clientset,
	namespace, podCommand string) (*resource.NetworkPod, error) {
	scheduling := resource.PodScheduling{ScheduleOn: resource.GatewayNode, Networking: resource.HostNetworking}
	return spawnPod(clientSet, scheduling, "validate-sniffer",
		namespace, podCommand)
}

func spawnSnifferPodOnNode(clientSet *kubernetes.Clientset,
	nodeName, namespace, podCommand string) (*resource.NetworkPod, error) {
	scheduling := resource.PodScheduling{ScheduleOn: resource.CustomNode, NodeName: nodeName,
		Networking: resource.HostNetworking}
	return spawnPod(clientSet, scheduling, "validate-sniffer",
		namespace, podCommand)
}

func spawnClientPodOnNonGatewayNode(clientSet *kubernetes.Clientset,
	namespace, podCommand string) (*resource.NetworkPod, error) {
	scheduling := resource.PodScheduling{ScheduleOn: resource.NonGatewayNode, Networking: resource.PodNetworking}
	return spawnPod(clientSet, scheduling, "validate-client",
		namespace, podCommand)
}

func spawnPod(clientSet *kubernetes.Clientset, scheduling resource.PodScheduling, podName, namespace,
	podCommand string) (*resource.NetworkPod, error) {
	pod, err := resource.SchedulePod(&resource.PodConfig{
		Name:       podName,
		ClientSet:  clientSet,
		Scheduling: scheduling,
		Namespace:  namespace,
		Command:    podCommand,
	})

	if err != nil {
		return nil, err
	}
	return pod, nil
}
