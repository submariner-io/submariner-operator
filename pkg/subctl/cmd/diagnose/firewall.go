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
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/pods"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	subv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type TargetPort int

const (
	TunnelPort TargetPort = iota
	NatDiscoveryPort
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

func spawnSnifferPodOnGatewayNode(client kubernetes.Interface, namespace, podCommand string) (*pods.Scheduled, error) {
	scheduling := pods.Scheduling{ScheduleOn: pods.GatewayNode, Networking: pods.HostNetworking}
	return spawnPod(client, scheduling, "validate-sniffer", namespace, podCommand)
}

func spawnSnifferPodOnNode(client kubernetes.Interface, nodeName, namespace, podCommand string) (*pods.Scheduled, error) {
	scheduling := pods.Scheduling{
		ScheduleOn: pods.CustomNode, NodeName: nodeName,
		Networking: pods.HostNetworking,
	}

	return spawnPod(client, scheduling, "validate-sniffer", namespace, podCommand)
}

func spawnClientPodOnNonGatewayNode(client kubernetes.Interface, namespace, podCommand string) (*pods.Scheduled, error) {
	scheduling := pods.Scheduling{ScheduleOn: pods.NonGatewayNode, Networking: pods.PodNetworking}
	return spawnPod(client, scheduling, "validate-client", namespace, podCommand)
}

func spawnClientPodOnNonGWNodeWithHostNwk(client kubernetes.Interface, namespace, podCommand string) (*pods.Scheduled, error) {
	scheduling := pods.Scheduling{ScheduleOn: pods.NonGatewayNode, Networking: pods.HostNetworking}
	return spawnPod(client, scheduling, "validate-client", namespace, podCommand)
}

func spawnPod(client kubernetes.Interface, scheduling pods.Scheduling, podName, namespace,
	podCommand string) (*pods.Scheduled, error) {
	pod, err := pods.Schedule(&pods.Config{
		Name:       podName,
		ClientSet:  client,
		Scheduling: scheduling,
		Namespace:  namespace,
		Command:    podCommand,
	})
	if err != nil {
		return nil, errors.Wrap(err, "error scheduling pod")
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

	for i := range nodes.Items {
		node := &nodes.Items[i]
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

		defer sPod.Delete()

		if err = sPod.AwaitCompletion(); err != nil {
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

	for i := range endpoints.Items {
		if endpoints.Items[i].Spec.ClusterID == cluster.Submariner.Spec.ClusterID {
			return &endpoints.Items[i]
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

	for i := range endpoints.Items {
		if endpoints.Items[i].Spec.ClusterID != cluster.Submariner.Spec.ClusterID {
			return &endpoints.Items[i]
		}
	}

	status.EndWithFailure("Could not find any remote Endpoint in cluster %q", cluster.Name)

	return nil
}

func checkKubeconfigArgs(_ *cobra.Command, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("two kubeconfigs must be specified")
	}

	same, err := cmd.CompareFiles(args[0], args[1])
	if err != nil {
		return errors.Wrap(err, "error comparing the kubeconfig files")
	}

	if same {
		return fmt.Errorf("the specified kubeconfig files are the same")
	}

	return nil
}

func validateConnectivity(localCluster, remoteCluster *cmd.Cluster, targetPort TargetPort, status *cli.Status) bool {
	localEndpoint := getLocalEndpointResource(localCluster, status)
	if localEndpoint == nil {
		return false
	}

	gwNodeName := getActiveGatewayNodeName(localCluster, localEndpoint.Spec.Hostname, status)
	if gwNodeName == "" {
		return false
	}

	destPort, err := getTargetPort(localCluster.Submariner, localEndpoint, targetPort)
	if err != nil {
		status.EndWithFailure(fmt.Sprintf("Could not determine the target port: %v", err))
		return false
	}

	clientMessage := string(uuid.NewUUID())[0:8]
	podCommand := fmt.Sprintf("timeout %d tcpdump -ln -Q in -A -s 100 -i any udp and dst port %d | grep '%s'",
		validationTimeout, destPort, clientMessage)

	sPod, err := spawnSnifferPodOnNode(localCluster.KubeClient, gwNodeName, podNamespace, podCommand)
	if err != nil {
		status.EndWithFailure("Error spawning the sniffer pod on the Gateway node: %v", err)
		return false
	}

	defer sPod.Delete()

	gatewayPodIP := getGatewayIP(remoteCluster, localCluster.Name, status)
	if gatewayPodIP == "" {
		status.EndWithFailure("Error retrieving the gateway IP of cluster %q", localCluster.Name)
		return false
	}

	podCommand = fmt.Sprintf("for x in $(seq 1000); do echo %s; done | for i in $(seq 5);"+
		" do timeout 2 nc -n -p %s -u %s %d; done", clientMessage, clientSourcePort, gatewayPodIP, destPort)

	// Spawn the pod on the nonGateway node. If we spawn the pod on Gateway node, the tunnel process can
	// sometimes drop the udp traffic from client pod until the tunnels are properly setup.
	cPod, err := spawnClientPodOnNonGWNodeWithHostNwk(remoteCluster.KubeClient, podNamespace, podCommand)
	if err != nil {
		status.EndWithFailure("Error spawning the client pod on non-Gateway node of cluster %q: %v",
			remoteCluster.Name, err)
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

	if !strings.Contains(sPod.PodOutput, clientMessage) {
		status.EndWithFailure("The tcpdump output from the sniffer pod does not include the message"+
			" sent from client pod. Please check that your firewall configuration allows UDP/%d traffic"+
			" on the %q node.", destPort, localEndpoint.Spec.Hostname)

		return false
	}

	return true
}

func getTargetPort(submariner *v1alpha1.Submariner, endpoint *subv1.Endpoint, tgtport TargetPort) (int32, error) {
	var targetPort int32
	var err error

	switch endpoint.Spec.Backend {
	case "libreswan", "wireguard", "vxlan":
		if tgtport == TunnelPort {
			targetPort, err = endpoint.Spec.GetBackendPort(subv1.UDPPortConfig, int32(submariner.Spec.CeIPSecNATTPort))
			if err != nil {
				return 0, fmt.Errorf("error reading tunnel port: %w", err)
			}
		} else if tgtport == NatDiscoveryPort {
			targetPort, err = endpoint.Spec.GetBackendPort(subv1.NATTDiscoveryPortConfig, 4490)
			if err != nil {
				return 0, fmt.Errorf("error reading nat-discovery port: %w", err)
			}
		}

		return targetPort, nil
	default:
		return 0, fmt.Errorf("could not determine the target port for cable driver %q", endpoint.Spec.Backend)
	}
}

func getGatewayIP(cluster *cmd.Cluster, localClusterID string, status *cli.Status) string {
	gateways, err := cluster.GetGateways()
	if err != nil {
		status.EndWithFailure("Error retrieving gateways from cluster %q: %v", cluster.Name, err)
		return ""
	}

	if len(gateways) == 0 {
		status.EndWithFailure("There are no gateways detected on cluster %q", cluster.Name)
		return ""
	}

	for i := range gateways {
		gw := &gateways[i]
		if gw.Status.HAStatus != subv1.HAStatusActive {
			continue
		}

		for j := range gw.Status.Connections {
			conn := &gw.Status.Connections[j]
			if conn.Endpoint.ClusterID == localClusterID {
				if conn.UsingIP != "" {
					return conn.UsingIP
				}

				if conn.Endpoint.NATEnabled {
					return conn.Endpoint.PublicIP
				}

				return conn.Endpoint.PrivateIP
			}
		}
	}

	status.EndWithFailure("The gateway on cluster %q does not have an active connection to cluster %q",
		cluster.Name, localClusterID)

	return ""
}

func newCluster(cfg *rest.Config) *cmd.Cluster {
	cluster, errMsg := cmd.NewCluster(cfg, "")
	if cluster == nil {
		utils.ExitWithErrorMsg(errMsg)
	}

	if cluster.Submariner == nil {
		utils.ExitWithErrorMsg(cmd.SubmMissingMessage)
	}

	cluster.Name = cluster.Submariner.Spec.ClusterID

	return cluster
}

func getClusterDetails(args []string) (*cmd.Cluster, *cmd.Cluster) {
	localProducer := restconfig.NewProducerFrom(args[0], "")
	localCfg, err := localProducer.ForCluster()
	utils.ExitOnError("The provided local kubeconfig is invalid", err)

	remoteProducer := restconfig.NewProducerFrom(args[1], "")
	remoteCfg, err := remoteProducer.ForCluster()
	utils.ExitOnError("The provided remote kubeconfig is invalid", err)

	localCluster := newCluster(localCfg)
	remoteCluster := newCluster(remoteCfg)

	return localCluster, remoteCluster
}
