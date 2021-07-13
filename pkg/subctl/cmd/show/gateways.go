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
package show

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
)

type gatewayStatus struct {
	node     string
	haStatus submv1.HAStatus
	summary  string
}

func init() {
	showCmd.AddCommand(&cobra.Command{
		Use:     "gateways",
		Short:   "Show submariner gateway summary information",
		Long:    `This command shows summary information about the submariner gateways in a cluster.`,
		PreRunE: cmd.CheckVersionMismatch,
		Run: func(command *cobra.Command, args []string) {
			cmd.ExecuteMultiCluster(showGateways)
		},
	})
}

func getGatewaysStatus(cluster *cmd.Cluster) bool {
	status := cli.NewStatus()
	status.Start("Showing Gateways")

	gateways, err := cluster.GetGateways()

	if err != nil {
		status.EndWithFailure("Error retrieving gateways: %v", err)
		return false
	}

	if len(gateways.Items) == 0 {
		status.EndWithFailure("There are no gateways detected")
		return false
	}

	var gwStatus = make([]gatewayStatus, 0, len(gateways.Items))
	for _, gateway := range gateways.Items {
		haStatus := gateway.Status.HAStatus
		enpoint := gateway.Status.LocalEndpoint.Hostname
		totalConnections := len(gateway.Status.Connections)
		countConnected := 0
		for _, connection := range gateway.Status.Connections {
			if connection.Status == submv1.Connected {
				countConnected += 1
			}
		}

		var summary string
		if gateway.Status.StatusFailure != "" {
			summary = gateway.Status.StatusFailure
		} else if totalConnections == 0 {
			summary = "There are no connections"
		} else if totalConnections == countConnected {
			summary = fmt.Sprintf("All connections (%d) are established", totalConnections)
		} else {
			summary = fmt.Sprintf("%d connections out of %d are established", countConnected, totalConnections)
		}
		gwStatus = append(gwStatus,
			gatewayStatus{
				node:     enpoint,
				haStatus: haStatus,
				summary:  summary,
			})
	}
	if len(gwStatus) == 0 {
		status.EndWithFailure("No Gateways found")
		return false
	}
	status.End(cli.Success)
	printGateways(gwStatus)
	return true
}

func showGateways(cluster *cmd.Cluster) bool {
	status := cli.NewStatus()

	if cluster.Submariner == nil {
		status.Start(cmd.SubmMissingMessage)
		status.End(cli.Warning)
		return true
	}

	return getGatewaysStatus(cluster)
}

func printGateways(gateways []gatewayStatus) {
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
