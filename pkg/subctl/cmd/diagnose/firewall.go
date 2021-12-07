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
	"context"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
	"github.com/submariner-io/submariner-operator/pkg/subctl/resource"
	subv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var diagnoseFirewallConfigCmd = &cobra.Command{
	Use:   "firewall",
	Short: "Check the firewall configuration",
	Long:  "This command checks if the firewall is configured as per Submariner pre-requisites.",
}

var validationTimeout uint

func addDiagnoseFWConfigFlags(command *cobra.Command) {
	command.Flags().UintVar(&validationTimeout, "validation-timeout", 90,
		"timeout in seconds while validating the connection attempt")
	addNamespaceFlag(command)
}

func init() {
	diagnoseCmd.AddCommand(diagnoseFirewallConfigCmd)
}

func spawnSnifferPodOnGatewayNode(client kubernetes.Interface, namespace, podCommand string) (*resource.NetworkPod, error) {
	scheduling := resource.PodScheduling{ScheduleOn: resource.GatewayNode, Networking: resource.HostNetworking}
	return spawnPod(client, scheduling, "validate-sniffer", namespace, podCommand)
}

func spawnSnifferPodOnNode(client kubernetes.Interface, nodeName, namespace, podCommand string) (*resource.NetworkPod, error) {
	scheduling := resource.PodScheduling{ScheduleOn: resource.CustomNode, NodeName: nodeName,
		Networking: resource.HostNetworking}
	return spawnPod(client, scheduling, "validate-sniffer", namespace, podCommand)
}

func spawnClientPodOnNonGatewayNode(client kubernetes.Interface, namespace, podCommand string) (*resource.NetworkPod, error) {
	scheduling := resource.PodScheduling{ScheduleOn: resource.NonGatewayNode, Networking: resource.PodNetworking}
	return spawnPod(client, scheduling, "validate-client", namespace, podCommand)
}

func spawnPod(client kubernetes.Interface, scheduling resource.PodScheduling, podName, namespace,
	podCommand string) (*resource.NetworkPod, error) {
	pod, err := resource.SchedulePod(&resource.PodConfig{
		Name:       podName,
		ClientSet:  client,
		Scheduling: scheduling,
		Namespace:  namespace,
		Command:    podCommand,
	})

	if err != nil {
		return nil, err
	}

	return pod, nil
}

func getActiveGatewayNodeName(cluster *cmd.Cluster, hostname string, status *cli.Status) string {
	nodes, err := cluster.KubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: "submariner.io/gateway=true",
	})
	if err != nil {
		status.EndWithFailure("Error obtaining the Gateway Nodes in cluster %q: %v", cluster.Name, err)
		return ""
	}

	for _, node := range nodes.Items {
		if node.Name == hostname {
			return hostname
		}

		// On some platforms, the nodeName does not match with the hostname.
		// Submariner Endpoint stores the hostname info in the endpoint and not the nodeName. So, we spawn a
		// tiny pod to read the hostname and return the corresponding node.
		sPod, err := spawnSnifferPodOnNode(cluster.KubeClient, node.Name, "default", "hostname")
		if err != nil {
			status.EndWithFailure("Error spawning the sniffer pod on the node %q: %v", node.Name, err)
			return ""
		}

		defer sPod.DeletePod()

		if err = sPod.AwaitPodCompletion(); err != nil {
			status.EndWithFailure("Error waiting for the sniffer pod to finish its execution on node %q: %v", node.Name, err)
			return ""
		}

		if sPod.PodOutput[:len(sPod.PodOutput)-1] == hostname {
			return node.Name
		}
	}

	status.EndWithFailure("Could not find the active Gateway node %q in local cluster in cluster %q",
		hostname, cluster.Name)
	return ""
}

func getLocalEndpointResource(cluster *cmd.Cluster, status *cli.Status) *subv1.Endpoint {
	endpoints, err := cluster.SubmClient.SubmarinerV1().Endpoints(cmd.OperatorNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		status.EndWithFailure("Error obtaining the Endpoints in cluster %q: %v", cluster.Name, err)
		return nil
	}

	for _, endpoint := range endpoints.Items {
		if endpoint.Spec.ClusterID == cluster.Submariner.Spec.ClusterID {
			return &endpoint
		}
	}

	status.EndWithFailure("Could not find the local Endpoint in cluster %q", cluster.Name)
	return nil
}

func getAnyRemoteEndpointResource(cluster *cmd.Cluster, status *cli.Status) *subv1.Endpoint {
	endpoints, err := cluster.SubmClient.SubmarinerV1().Endpoints(cmd.OperatorNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		status.EndWithFailure("Error obtaining the Endpoints in cluster %q: %v", cluster.Name, err)
		return nil
	}

	for _, endpoint := range endpoints.Items {
		if endpoint.Spec.ClusterID != cluster.Submariner.Spec.ClusterID {
			return &endpoint
		}
	}

	status.EndWithFailure("Could not find any remote Endpoint in cluster %q", cluster.Name)
	return nil
}
