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
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	subv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/rest"
)

const (
	clientSourcePort = "9898"
)

func init() {
	command := &cobra.Command{
		Use:   "inter-cluster <localkubeconfig> <remotekubeconfig>",
		Short: "Check firewall access to setup tunnels between the Gateway node",
		Long:  "This command checks if the firewall configuration allows tunnels to be configured on the Gateway nodes.",
		Args: func(command *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("two kubeconfigs must be specified")
			}

			same, err := cmd.CompareFiles(args[0], args[1])
			if err != nil {
				return err // nolint:wrapcheck // No need to wrap here
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
	localProducer := restconfig.NewProducerFrom(args[0], "")
	localCfg, err := localProducer.ForCluster()
	utils.ExitOnError("The provided local kubeconfig is invalid", err)

	remoteProducer := restconfig.NewProducerFrom(args[1], "")
	remoteCfg, err := remoteProducer.ForCluster()
	utils.ExitOnError("The provided remote kubeconfig is invalid", err)

	if !validateTunnelConfigAcrossClusters(localCfg.Config, remoteCfg.Config) {
		os.Exit(1)
	}
}

func validateTunnelConfigAcrossClusters(localCfg, remoteCfg *rest.Config) bool {
	localCluster := newCluster(localCfg)

	localCluster.Name = localCluster.Submariner.Spec.ClusterID

	remoteCluster := newCluster(remoteCfg)

	remoteCluster.Name = remoteCluster.Submariner.Spec.ClusterID

	status := cli.NewStatus()
	status.Start(fmt.Sprintf("Checking if tunnels can be setup on the gateway node of cluster %q", localCluster.Name))

	if isClusterSingleNode(remoteCluster, status) {
		// Skip the check if it's a single node cluster
		return true
	}

	localEndpoint := getLocalEndpointResource(localCluster, status)
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

	defer sPod.Delete()

	gatewayPodIP := getGatewayIP(remoteCluster, localCluster.Name, status)
	if gatewayPodIP == "" {
		status.EndWithFailure("Error retrieving the gateway IP of cluster %q", localCluster.Name)
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
			" on the %q node.", tunnelPort, localEndpoint.Spec.Hostname)

		return false
	}

	status.EndWithSuccess("Tunnels can be established on the gateway node")

	return true
}

func newCluster(cfg *rest.Config) *cmd.Cluster {
	cluster, errMsg := cmd.NewCluster(cfg, "")
	if cluster == nil {
		utils.ExitWithErrorMsg(errMsg)
	}

	if cluster.Submariner == nil {
		utils.ExitWithErrorMsg(cmd.SubmMissingMessage)
	}

	return cluster
}

func getTunnelPort(submariner *v1alpha1.Submariner, endpoint *subv1.Endpoint, status *cli.Status) (int32, bool) {
	var tunnelPort int32
	var err error

	switch endpoint.Spec.Backend {
	case "libreswan", "wireguard", "vxlan":
		tunnelPort, err = endpoint.Spec.GetBackendPort(subv1.UDPPortConfig, int32(submariner.Spec.CeIPSecNATTPort))
		if err != nil {
			status.EndWithFailure(fmt.Sprintf("Error reading tunnel port: %v", err))
			return tunnelPort, false
		}

		return tunnelPort, true
	default:
		status.EndWithFailure(fmt.Sprintf("Could not determine the tunnel port for cable driver %q",
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
