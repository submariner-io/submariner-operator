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
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils/restconfig"
	subv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/rest"
)

const (
	clientSourcePort = "9898"
)

func init() {
	command := &cobra.Command{
		Use:   "tunnel <localkubeconfig> <remotekubeconfig>",
		Short: "Check firewall access to Gateway node tunnels",
		Long:  "This command checks if the firewall configuration allows tunnels to be configured on the Gateway nodes.",
		Args: func(command *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("two kubeconfigs must be specified")
			}

			same, err := cmd.CompareFiles(args[0], args[1])
			if err != nil {
				return err
			}

			if same {
				return fmt.Errorf("the specified kubeconfig files are the same")
			}

			return nil
		},
		Run: validateTunnelConfig,
	}

	addDiagnoseFWConfigFlags(command)
	addVerboseFlag(command)

	diagnoseFirewallConfigCmd.AddCommand(command)
}

func validateTunnelConfig(command *cobra.Command, args []string) {
	localCfg, err := restconfig.ForCluster(args[0], "")
	utils.ExitOnError("The provided local kubeconfig is invalid", err)

	remoteCfg, err := restconfig.ForCluster(args[1], "")
	utils.ExitOnError("The provided remote kubeconfig is invalid", err)

	if !validateTunnelConfigAcrossClusters(localCfg, remoteCfg) {
		os.Exit(1)
	}
}

func validateTunnelConfigAcrossClusters(localCfg, remoteCfg *rest.Config) bool {
	localCluster, errMsg := cmd.NewCluster(localCfg, "")
	if localCluster == nil {
		utils.ExitWithErrorMsg(errMsg)
	}

	if localCluster.Submariner == nil {
		utils.ExitWithErrorMsg(cmd.SubmMissingMessage)
	}

	localCluster.Name = localCluster.Submariner.Spec.ClusterID

	remoteCluster, errMsg := cmd.NewCluster(remoteCfg, "")
	if remoteCluster == nil {
		utils.ExitWithErrorMsg(errMsg)
	}

	if remoteCluster.Submariner == nil {
		utils.ExitWithErrorMsg(cmd.SubmMissingMessage)
	}

	remoteCluster.Name = remoteCluster.Submariner.Spec.ClusterID

	status := cli.NewStatus()
	status.Start(fmt.Sprintf("Checking if tunnels can be setup on the gateway node of cluster %q", localCluster.Name))

	localEndpoint := getEndpointResource(localCluster, status)
	if localEndpoint == nil {
		return false
	}

	gwNodeName := getActiveGatewayNodeName(localCluster, localEndpoint.Spec.Hostname, status)
	if gwNodeName == "" {
		return false
	}

	tunnelPort, ok := getTunnelPort(localCluster.Submariner, localEndpoint, status)
	if !ok {
		return false
	}

	clientMessage := string(uuid.NewUUID())[0:8]
	podCommand := fmt.Sprintf("timeout %d tcpdump -ln -Q in -A -s 100 -i any udp and dst port %d | grep '%s'",
		validationTimeout, tunnelPort, clientMessage)
	sPod, err := spawnSnifferPodOnNode(localCluster.KubeClient, gwNodeName, podNamespace, podCommand)
	if err != nil {
		status.EndWithFailure("Error spawning the sniffer pod on the Gateway node: %v", err)
		return false
	}

	defer sPod.DeletePod()

	gatewayPodIP := getGatewayIP(remoteCluster, localCluster.Name, status)
	if gatewayPodIP == "" {
		return false
	}

	podCommand = fmt.Sprintf("for x in $(seq 1000); do echo %s; done | for i in $(seq 5);"+
		" do timeout 2 nc -n -p %s -u %s %d; done", clientMessage, clientSourcePort, gatewayPodIP, tunnelPort)

	// Spawn the pod on the nonGateway node. If we spawn the pod on Gateway node, the tunnel process can
	// sometimes drop the udp traffic from client pod until the tunnels are properly setup.
	cPod, err := spawnClientPodOnNonGatewayNode(remoteCluster.KubeClient, podNamespace, podCommand)
	if err != nil {
		status.EndWithFailure("Error spawning the client pod on non-Gateway node of cluster %q: %v",
			remoteCluster.Name, err)
		return false
	}

	defer cPod.DeletePod()

	if err = cPod.AwaitPodCompletion(); err != nil {
		status.EndWithFailure("Error waiting for the client pod to finish its execution: %v", err)
		return false
	}

	if err = sPod.AwaitPodCompletion(); err != nil {
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
			" on the %q node.", tunnelPort, localEndpoint.Spec.Hostname)
		return false
	}

	status.EndWithSuccess("Tunnels can be established on the gateway node")

	return true
}

func getEndpointResource(cluster *cmd.Cluster, status *cli.Status) *subv1.Endpoint {
	endpoints, err := cluster.SubmClient.SubmarinerV1().Endpoints(cmd.OperatorNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		status.EndWithFailure("Error obtaining the Endpoints in cluster %q: %v", cluster.Name, err)
		return nil
	}

	for _, endpoint := range endpoints.Items {
		if endpoint.Spec.ClusterID == cluster.Name {
			return &endpoint
		}
	}

	status.EndWithFailure("Could not find the local Endpoint in cluster %q", cluster.Name)
	return nil
}

func getActiveGatewayNodeName(cluster *cmd.Cluster, hostname string, status *cli.Status) string {
	nodes, err := cluster.KubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		status.EndWithFailure("Error obtaining the Nodes in cluster %q: %v", cluster.Name, err)
		return ""
	}

	for _, node := range nodes.Items {
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeHostName {
				if strings.HasPrefix(addr.Address, hostname) {
					return node.Name
				}
			}
		}
	}

	status.EndWithFailure("Could not find the active Gateway node %q in local cluster in cluster %q",
		hostname, cluster.Name)
	return ""
}

func getTunnelPort(submariner *v1alpha1.Submariner, endpoint *subv1.Endpoint, status *cli.Status) (int32, bool) {
	var tunnelPort int32
	var err error

	switch endpoint.Spec.Backend {
	case "libreswan", "wireguard":
		tunnelPort, err = endpoint.Spec.GetBackendPort(subv1.UDPPortConfig, int32(submariner.Spec.CeIPSecNATTPort))
		if err != nil {
			status.QueueWarningMessage(fmt.Sprintf("Error reading tunnel port: %v", err))
		}

		return tunnelPort, true
	default:
		status.QueueFailureMessage(fmt.Sprintf("Could not determine the tunnel port for cable driver %q",
			endpoint.Spec.Backend))
		return tunnelPort, false
	}
}

func getGatewayIP(cluster *cmd.Cluster, localClusterID string, status *cli.Status) string {
	gateways, err := cluster.GetGateways()
	if err != nil {
		status.EndWithFailure("Error retrieving gateways from cluster %q: %v", cluster.Name, err)
		return ""
	}

	if gateways == nil {
		status.EndWithFailure("There are no gateways detected on cluster %q", cluster.Name)
		return ""
	}

	for i := range gateways.Items {
		gw := &gateways.Items[i]
		if gw.Status.HAStatus != subv1.HAStatusActive {
			continue
		}

		for _, conn := range gw.Status.Connections {
			if conn.Endpoint.ClusterID == localClusterID {
				return conn.UsingIP
			}
		}
	}

	status.EndWithFailure("The gateway on cluster %q does not have an active connection to cluster %q",
		cluster.Name, localClusterID)
	return ""
}
