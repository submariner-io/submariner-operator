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

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
)

func init() {
	diagnoseCmd.AddCommand(&cobra.Command{
		Use:   "connections",
		Short: "Check the Gateway connections",
		Long:  "This command checks that the Gateway connections to other clusters are all established",
		Run: func(command *cobra.Command, args []string) {
			cmd.ExecuteMultiCluster(checkConnections)
		},
	})
}

func checkConnections(cluster *cmd.Cluster) bool {
	status := cli.NewStatus()

	if cluster.Submariner == nil {
		status.Start(cmd.SubmMissingMessage)
		status.End(cli.Warning)

		return true
	}

	status.Start("Checking gateway connections")

	gateways, err := cluster.GetGateways()
	if err != nil {
		status.EndWithFailure("Error retrieving gateways: %v", err)
		return false
	}

	if len(gateways) == 0 {
		status.EndWithFailure("There are no gateways detected")
		return false
	}

	foundActive := false

	for i := range gateways {
		gateway := &gateways[i]
		if gateway.Status.HAStatus != submv1.HAStatusActive {
			continue
		}

		foundActive = true

		if len(gateway.Status.Connections) == 0 {
			status.QueueFailureMessage(fmt.Sprintf("There are no active connections on gateway %q", gateway.Name))
		}

		for j := range gateway.Status.Connections {
			connection := &gateway.Status.Connections[j]
			if connection.Status == submv1.Connecting {
				status.QueueFailureMessage(fmt.Sprintf("Connection to cluster %q is in progress", connection.Endpoint.ClusterID))
			} else if connection.Status == submv1.ConnectionError {
				status.QueueFailureMessage(fmt.Sprintf("Connection to cluster %q is not established", connection.Endpoint.ClusterID))
			}
		}
	}

	if !foundActive {
		status.QueueFailureMessage("No active gateway was found")
	}

	if status.HasFailureMessages() {
		status.End(cli.Failure)
		return false
	}

	status.EndWithSuccess("All connections are established")

	return true
}
