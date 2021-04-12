/*
© 2021 Red Hat, Inc. and others

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
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"

	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
)

type connectionStatus struct {
	gateway     string
	cluster     string
	remoteIP    string
	usingNAT    string
	cableDriver string
	subnets     string
	status      submv1.ConnectionStatus
}

var showConnectionsCmd = &cobra.Command{
	Use:     "connections",
	Short:   "Show cluster connectivity information",
	Long:    `This command shows information about submariner endpoint connections with other clusters.`,
	PreRunE: checkVersionMismatch,
	Run:     showConnections,
}

func init() {
	showCmd.AddCommand(showConnectionsCmd)
}

func getConnectionsStatus(submariner *v1alpha1.Submariner) []connectionStatus {
	var status []connectionStatus

	gateways := submariner.Status.Gateways
	if gateways == nil {
		exitWithErrorMsg("No endpoints found")
	}

	for _, gateway := range *gateways {
		for _, connection := range gateway.Connections {
			subnets := strings.Join(connection.Endpoint.Subnets, ", ")

			ip, nat := remoteIPAndNATForConnection(connection)
			status = append(status, connectionStatus{
				gateway:     connection.Endpoint.Hostname,
				cluster:     connection.Endpoint.ClusterID,
				remoteIP:    ip,
				usingNAT:    nat,
				cableDriver: connection.Endpoint.Backend,
				subnets:     subnets,
				status:      connection.Status,
			})
		}
	}

	return status
}

func remoteIPAndNATForConnection(connection submv1.Connection) (string, string) {
	usingNAT := "no"
	if connection.UsingIP != "" {
		if connection.UsingNAT {
			usingNAT = "yes"
		}
		return connection.UsingIP, usingNAT
	}

	if connection.Endpoint.NATEnabled {
		return connection.Endpoint.PublicIP, "yes"
	} else {
		return connection.Endpoint.PrivateIP, "no"
	}
}

func showConnections(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContexts)
	exitOnError("Error getting REST config for cluster", err)
	for _, item := range configs {
		fmt.Println()
		fmt.Printf("Showing information for cluster %q:\n", item.clusterName)
		submariner := getSubmarinerResource(item.config)

		if submariner == nil {
			fmt.Println(submMissingMessage)
		} else {
			showConnectionsFor(submariner)
		}
	}
}

func showConnectionsFor(submariner *v1alpha1.Submariner) {
	status := getConnectionsStatus(submariner)
	printConnections(status)
}

func printConnections(connections []connectionStatus) {
	if len(connections) == 0 {
		fmt.Println("No resources found.")
		return
	}

	template := "%-32.31s%-24.23s%-16.15s%-5.3s%-20.19s%-40.39s%-16.15s\n"
	fmt.Printf(template, "GATEWAY", "CLUSTER", "REMOTE IP", "NAT", "CABLE DRIVER", "SUBNETS", "STATUS")

	for _, item := range connections {
		fmt.Printf(
			template,
			item.gateway,
			item.cluster,
			item.remoteIP,
			item.usingNAT,
			item.cableDriver,
			item.subnets,
			item.status)
	}
}
