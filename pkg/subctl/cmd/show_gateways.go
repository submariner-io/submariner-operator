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

	"github.com/spf13/cobra"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"

	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
)

type gatewayStatus struct {
	node     string
	haStatus submv1.HAStatus
	summary  string
}

var showGatewaysCmd = &cobra.Command{
	Use:   "gateways",
	Short: "Show submariner gateway summary information",
	Long:  `This command shows summary information about the submariner gateways in a cluster.`,
	Run:   showGateways,
}

func init() {
	showCmd.AddCommand(showGatewaysCmd)
}

func getGatewaysStatus(submariner *v1alpha1.Submariner) []gatewayStatus {
	var status []gatewayStatus

	gateways := submariner.Status.Gateways
	if gateways == nil {
		exitWithErrorMsg("no gateways found")
	}

	for _, gateway := range *gateways {
		haStatus := gateway.HAStatus
		enpoint := gateway.LocalEndpoint.Hostname
		totalConnections := len(gateway.Connections)
		countConnected := 0
		for _, connection := range gateway.Connections {
			if connection.Status == submv1.Connected {
				countConnected += 1
			}
		}

		var summary string
		if gateway.StatusFailure != "" {
			summary = gateway.StatusFailure
		} else if totalConnections == 0 {
			summary = "There are no connections"
		} else if totalConnections == countConnected {
			summary = fmt.Sprintf("All connections (%d) are established", totalConnections)
		} else {
			summary = fmt.Sprintf("%d connections out of %d are established", countConnected, totalConnections)
		}
		status = append(status,
			gatewayStatus{
				node:     enpoint,
				haStatus: haStatus,
				summary:  summary,
			})
	}

	return status
}

func showGateways(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContext)
	exitOnError("Error getting REST config for cluster", err)

	for _, item := range configs {
		fmt.Println()
		fmt.Printf("Showing information for cluster %q:\n", item.clusterName)
		submariner := getSubmarinerResource(item.config)

		if submariner == nil {
			fmt.Println(submMissingMessage)
		} else {
			showGatewaysFor(submariner)
		}
	}
}

func showGatewaysFor(submariner *v1alpha1.Submariner) {
	status := getGatewaysStatus(submariner)
	printGateways(status)
}

func printGateways(gateways []gatewayStatus) {
	if len(gateways) == 0 {
		fmt.Println("No resources found.")
		return
	}

	template := "%-32.31s%-16s%-32s\n"
	fmt.Printf(template, "NODE", "HA STATUS", "SUMMARY")

	for _, item := range gateways {
		fmt.Printf(
			template,
			item.node,
			item.haStatus,
			item.summary)
	}
}
