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
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	subv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"

	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
)

const (
	ClientSourcePort = "9898"
)

var verboseOutput bool

var validateTunnelCmd = &cobra.Command{
	Use:   "tunnel <localkubeconfig> <remotekubeconfig>",
	Short: "Check firewall access to Gateway node tunnels",
	Long:  "This command checks if the firewall configuration allows tunnels to be configured on the Gateway nodes.",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("two kubeconfigs must be specified")
		}
		same, err := compareFiles(args[0], args[1])
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

func init() {
	addValidateFWConfigFlags(validateTunnelCmd)
	validateTunnelCmd.Flags().BoolVar(&verboseOutput, "verbose", false,
		"produce verbose logs during validation")
	validateFirewallConfigCmd.AddCommand(validateTunnelCmd)
}

func validateTunnelConfig(cmd *cobra.Command, args []string) {
	localCfg, err := getRestConfig(args[0], "")
	exitOnError("The provided local kubeconfig is invalid", err)

	remoteCfg, err := getRestConfig(args[1], "")
	exitOnError("The provided remote kubeconfig is invalid", err)

	validationStatus := validateTunnelConfigAcrossClusters(localCfg, remoteCfg)
	status.End(status.ResultFromMessages())
	if !validationStatus {
		os.Exit(1)
	}
}

func validateTunnelConfigAcrossClusters(localCfg, remoteCfg *rest.Config) bool {
	lClientSet, err := kubernetes.NewForConfig(localCfg)
	exitOnError("Error creating API server client:: %s", err)

	submariner := getSubmarinerResource(localCfg)
	if submariner == nil {
		exitWithErrorMsg(submMissingMessage)
	}

	status.Start(fmt.Sprintf("Checking if tunnels can be setup on Gateway node of cluster %q.",
		submariner.Spec.ClusterID))

	localEndpoint := getLocalEndpointResource(localCfg, submariner.Spec.ClusterID)
	if localEndpoint == nil {
		status.QueueWarningMessage("Could not find the local cluster Endpoint")
		return false
	}

	gwNodeName := getActiveGatewayNodeName(lClientSet, localEndpoint.Spec.Hostname)
	if gwNodeName == "" {
		status.QueueWarningMessage("Could not find the active Gateway nodeName in local cluster")
		return false
	}

	tunnelPort, err := getTunnelPort(submariner, localEndpoint)
	if err != nil {
		return false
	}

	clientMessage := string(uuid.NewUUID())[0:8]
	podCommand := fmt.Sprintf("timeout %d tcpdump -ln -Q in -A -s 100 -i any udp and dst port %d | grep '%s'",
		validationTimeout, tunnelPort, clientMessage)
	sPod, err := spawnSnifferPodOnNode(lClientSet, gwNodeName, namespace, podCommand)
	if err != nil {
		status.QueueFailureMessage(fmt.Sprintf("Error while spawning the sniffer pod on the GatewayNode: %v", err))
		return false
	}
	defer sPod.DeletePod()

	gatewayPodIP := getGatewayIP(remoteCfg, submariner.Spec.ClusterID)
	if gatewayPodIP == "" {
		status.QueueWarningMessage("Gateway object on remote cluster does not have connection info to local cluster.")
		return false
	}

	rClientSet, err := kubernetes.NewForConfig(remoteCfg)
	if err != nil {
		message := fmt.Sprintf("Error creating API server client: %s", err)
		status.QueueFailureMessage(message)
		return false
	}

	podCommand = fmt.Sprintf("for x in $(seq 1000); do echo %s; done | for i in $(seq 5);"+
		" do timeout 2 nc -n -p %s -u %s %d; done", clientMessage, ClientSourcePort, gatewayPodIP, tunnelPort)
	// Spawn the pod on the nonGateway node. If we spawn the pod on Gateway node, the tunnel process can
	// sometimes drop the udp traffic from client pod until the tunnels are properly setup.
	cPod, err := spawnClientPodOnNonGatewayNode(rClientSet, namespace, podCommand)
	if err != nil {
		status.QueueFailureMessage(fmt.Sprintf("Error while spawning the client pod on non-Gateway node: %v", err))
		return false
	}

	defer cPod.DeletePod()
	if err = cPod.AwaitPodCompletion(); err != nil {
		status.QueueFailureMessage(fmt.Sprintf("Error while waiting for client pod to be finish its execution: %v", err))
		return false
	}

	if err = sPod.AwaitPodCompletion(); err != nil {
		status.QueueFailureMessage(fmt.Sprintf("Error while waiting for sniffer pod to be finish its execution: %v", err))
		return false
	}

	if verboseOutput {
		status.QueueSuccessMessage("tcpdump output from Sniffer Pod on Gateway node")
		status.QueueSuccessMessage(sPod.PodOutput)
	}

	return validateSnifferPodOutput(sPod.PodOutput, clientMessage, localEndpoint.Spec.Hostname, tunnelPort)
}

func getTunnelPort(submariner *v1alpha1.Submariner, endpoint *subv1.Endpoint) (int32, error) {
	var tunnelPort int32
	var err error
	switch endpoint.Spec.Backend {
	case "libreswan", "wireguard":
		tunnelPort, err = endpoint.Spec.GetBackendPort(subv1.UDPPortConfig, int32(submariner.Spec.CeIPSecNATTPort))
		if err != nil {
			status.QueueWarningMessage(fmt.Sprintf("Error reading tunnelPort: %v", err))
		}
		return tunnelPort, nil
	default:
		message := fmt.Sprintf("Could not determine the tunnel port for cable driver %q",
			endpoint.Spec.Backend)
		status.QueueFailureMessage(message)
		return tunnelPort, fmt.Errorf(message)
	}
}

func getGatewayIP(remoteCfg *rest.Config, localClusterID string) string {
	gateways := getGatewaysResource(remoteCfg)
	if gateways == nil {
		status.QueueWarningMessage("There are no gateways detected on the remote cluster.")
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

	return ""
}

func validateSnifferPodOutput(podOutput, clientMessage, hostname string, tunnelPort int32) bool {
	if !strings.Contains(podOutput, clientMessage) {
		message := fmt.Sprintf("The tcpdump output from the sniffer pod does not include the message"+
			" sent from client pod. Please check that your firewall configuration allows UDP/%d traffic"+
			" on the %q node.", tunnelPort, hostname)
		status.QueueFailureMessage(message)
		return false
	}

	status.QueueSuccessMessage("Tunnels can be successfully established on the Gateway node.")
	return true
}
