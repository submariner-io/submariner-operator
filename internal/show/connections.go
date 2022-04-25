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
	"strings"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
	"github.com/submariner-io/submariner-operator/pkg/subctl/table"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
)

type connectionStatus struct {
	gateway     string
	cluster     string
	remoteIP    string
	usingNAT    string
	cableDriver string
	subnets     string
	rtt         string
	status      submv1.ConnectionStatus
}

func init() {
	showCmd.AddCommand(&cobra.Command{
		Use:     "connections",
		Short:   "Show cluster connectivity information",
		Long:    `This command shows information about submariner endpoint connections with other clusters.`,
		PreRunE: restConfigProducer.CheckVersionMismatch,
		Run: func(command *cobra.Command, args []string) {
			cmd.ExecuteMultiCluster(restConfigProducer, showConnections)
		},
	})
}

func getConnectionsStatus(cluster *cmd.Cluster) bool {
	status := cli.NewStatus()
	status.Start("Showing Connections")

	gateways, err := cluster.GetGateways()
	if err != nil {
		status.EndWithFailure("Error retrieving gateways: %v", err)
		return false
	}

	if len(gateways) == 0 {
		status.EndWithFailure("There are no gateways detected")
		return false
	}

	var connStatus []interface{}

	for i := range gateways {
		gateway := &gateways[i]
		for i := range gateway.Status.Connections {
			connection := &gateway.Status.Connections[i]
			subnets := strings.Join(connection.Endpoint.Subnets, ", ")

			ip, nat := remoteIPAndNATForConnection(connection)

			connStatus = append(connStatus, connectionStatus{
				gateway:     connection.Endpoint.Hostname,
				cluster:     connection.Endpoint.ClusterID,
				remoteIP:    ip,
				usingNAT:    nat,
				cableDriver: connection.Endpoint.Backend,
				subnets:     subnets,
				status:      connection.Status,
				rtt:         getAverageRTTForConnection(connection),
			})
		}
	}

	if len(connStatus) == 0 {
		status.EndWithFailure("No connections found")
		return false
	}

	status.EndWith(cli.Success)
	connectionPrinter.Print(connStatus)

	return true
}

func getAverageRTTForConnection(connection *submv1.Connection) string {
	rtt := ""
	if connection.LatencyRTT != nil {
		rtt = connection.LatencyRTT.Average
	}

	return rtt
}

func remoteIPAndNATForConnection(connection *submv1.Connection) (string, string) {
	usingNAT := "no"

	if connection.UsingIP != "" {
		if connection.UsingNAT {
			usingNAT = "yes"
		}

		return connection.UsingIP, usingNAT
	}

	if connection.Endpoint.NATEnabled {
		return connection.Endpoint.PublicIP, "yes"
	}

	return connection.Endpoint.PrivateIP, "no"
}

func showConnections(cluster *cmd.Cluster) bool {
	status := cli.NewStatus()

	if cluster.Submariner == nil {
		status.Start(cmd.SubmMissingMessage)
		status.EndWith(cli.Warning)

		return true
	}

	return getConnectionsStatus(cluster)
}

var connectionPrinter = table.Printer{
	Headers: []table.Header{
		{Name: "GATEWAY", MaxLength: 31},
		{Name: "CLUSTER", MaxLength: 23},
		{Name: "REMOTE IP", MaxLength: 15},
		{Name: "NAT", MaxLength: 3},
		{Name: "CABLE DRIVER", MaxLength: 19},
		{Name: "SUBNETS", MaxLength: 39},
		{Name: "STATUS", MaxLength: 15},
		{Name: "RTT avg.", MaxLength: 12},
	},
	RowConverterFunc: func(obj interface{}) []string {
		item := obj.(connectionStatus)
		return []string{
			item.gateway, item.cluster, item.remoteIP, item.usingNAT, item.cableDriver,
			item.subnets, string(item.status), item.rtt,
		}
	},
}
