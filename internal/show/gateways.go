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

	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
)

type gatewayStatus struct {
	node     string
	haStatus submv1.HAStatus
	summary  string
}

func Gateways(newCluster *cluster.Info, status reporter.Interface) bool {
	if newCluster.Submariner == nil {
		status.Warning(constants.SubmMissingMessage)

		return true
	}

	return getGatewaysStatus(newCluster, status)
}

func getGatewaysStatus(newCluster *cluster.Info, status reporter.Interface) bool {
	status.Start("Showing Gateways")

	gateways, err := newCluster.GetGateways()
	if err != nil {
		status.Failure("Error retrieving gateways: %v", err)
		return false
	}

	if len(gateways) == 0 {
		status.Failure("There are no gateways detected")
		return false
	}

	gwStatus := make([]gatewayStatus, 0, len(gateways))

	for i := range gateways {
		gateway := &gateways[i]
		haStatus := gateway.Status.HAStatus
		enpoint := gateway.Status.LocalEndpoint.Hostname
		totalConnections := len(gateway.Status.Connections)
		countConnected := 0

		for i := range gateway.Status.Connections {
			if gateway.Status.Connections[i].Status == submv1.Connected {
				countConnected++
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
		status.Failure("No Gateways found")
		return false
	}

	status.End()
	printGateways(gwStatus)

	return true
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
