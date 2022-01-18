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

	"github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
	subv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

const (
	clientSourcePort = "9898"
)

// nolint:gocyclo // we are fine with cyclic complexity (more number of ifs)
func ValidateTunnelConfigAcrossClusters(localClientProducer, remoteClientProducer client.Producer, status reporter.Interface) bool {
	localCluster, localClusterFailed := newCluster(localClientProducer, status)
	remoteCluster, remoteClusterFailed := newCluster(remoteClientProducer, status)

	if localClusterFailed || remoteClusterFailed {
		return false
	}

	localCluster.Name = localCluster.Submariner.Spec.ClusterID
	remoteCluster.Name = remoteCluster.Submariner.Spec.ClusterID

	status.Start("Checking if tunnels can be setup on the gateway node of cluster %q", localCluster.Name)
	defer status.End()

	if isClusterSingleNode(remoteCluster, status) {
		// Skip the check if it's a single node cluster
		return true
	}

	localEndpoint, failed := getLocalEndpointResource(localCluster, status)
	if localEndpoint == nil || failed {
		return false
	}

	gwNodeName, failed := getActiveGatewayNodeName(localCluster, localEndpoint.Spec.Hostname, status)
	if gwNodeName == "" || failed {
		return false
	}

	tunnelPort, ok := getTunnelPort(localCluster.Submariner, localEndpoint, status)
	if !ok {
		return false
	}

	clientMessage := string(uuid.NewUUID())[0:8]
	podCommand := fmt.Sprintf("timeout %d tcpdump -ln -Q in -A -s 100 -i any udp and dst port %d | grep '%s'",
		ValidationTimeout, tunnelPort, clientMessage)

	sPod, err := spawnSnifferPodOnNode(localCluster.ClientProducer.ForKubernetes(), gwNodeName, KubeProxyPodNamespace, podCommand)
	if err != nil {
		status.Failure("Error spawning the sniffer pod on the Gateway node: %v", err)

		return false
	}

	defer sPod.Delete()

	gatewayPodIP := getGatewayIP(remoteCluster, localCluster.Name, status)
	if gatewayPodIP == "" {
		status.Failure("Error retrieving the gateway IP of cluster %q", localCluster.Name)

		return false
	}

	podCommand = fmt.Sprintf("for x in $(seq 1000); do echo %s; done | for i in $(seq 5);"+
		" do timeout 2 nc -n -p %s -u %s %d; done", clientMessage, clientSourcePort, gatewayPodIP, tunnelPort)

	// Spawn the pod on the nonGateway node. If we spawn the pod on Gateway node, the tunnel process can
	// sometimes drop the udp traffic from client pod until the tunnels are properly setup.
	cPod, err := spawnClientPodOnNonGatewayNode(remoteCluster.ClientProducer.ForKubernetes(), KubeProxyPodNamespace, podCommand)
	if err != nil {
		status.Failure("Error spawning the client pod on non-Gateway node of cluster %q: %v",
			remoteCluster.Name, err)

		return false
	}

	defer cPod.Delete()

	if err = cPod.AwaitCompletion(); err != nil {
		status.Failure("Error waiting for the client pod to finish its execution: %v", err)

		return false
	}

	if err = sPod.AwaitCompletion(); err != nil {
		status.Failure("Error waiting for the sniffer pod to finish its execution: %v", err)

		return false
	}

	if VerboseOutput {
		status.Success("tcpdump output from sniffer pod on Gateway node")
		status.Success(sPod.PodOutput)
	}

	if !strings.Contains(sPod.PodOutput, clientMessage) {
		status.Failure("The tcpdump output from the sniffer pod does not include the message"+
			" sent from client pod. Please check that your firewall configuration allows UDP/%d traffic"+
			" on the %q node.", tunnelPort, localEndpoint.Spec.Hostname)

		return false
	}

	status.Success("Tunnels can be established on the gateway node")

	return true
}

func newCluster(clientProducer client.Producer, status reporter.Interface) (*cluster.Info, bool) {
	clusterInfo, err := cluster.NewInfo("", clientProducer)
	if err != nil {
		status.Failure("Error initializing the cluster information: %v", err)
		return nil, true
	}

	if clusterInfo.Submariner == nil {
		status.Warning(constants.SubmMissingMessage)
		return nil, true
	}

	return clusterInfo, false
}

func getTunnelPort(submariner *v1alpha1.Submariner, endpoint *subv1.Endpoint, status reporter.Interface) (int32, bool) {
	var tunnelPort int32
	var err error

	switch endpoint.Spec.Backend {
	case "libreswan", "wireguard":
		tunnelPort, err = endpoint.Spec.GetBackendPort(subv1.UDPPortConfig, int32(submariner.Spec.CeIPSecNATTPort))
		if err != nil {
			status.Warning("Error reading tunnel port: %v", err)
		}

		return tunnelPort, true
	default:
		status.Failure("Could not determine the tunnel port for cable driver %q",
			endpoint.Spec.Backend)
		return tunnelPort, false
	}
}

func getGatewayIP(clusterInfo *cluster.Info, localClusterID string, status reporter.Interface) string {
	gateways, err := clusterInfo.GetGateways()
	if err != nil {
		status.Failure("Error retrieving gateways from cluster %q: %v", clusterInfo.Name, err)
		return ""
	}

	if len(gateways) == 0 {
		status.Failure("There are no gateways detected on cluster %q", clusterInfo.Name)
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

	status.Failure("The gateway on cluster %q does not have an active connection to cluster %q",
		clusterInfo.Name, localClusterID)

	return ""
}
