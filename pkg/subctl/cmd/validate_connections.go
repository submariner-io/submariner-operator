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

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
)

var validateConnectionsCmd = &cobra.Command{
	Use:   "connections",
	Short: "Validate the gateways connections",
	Long: "This command validates if the connections of the gateways are all " +
		"established",
	Run: validateConnections,
}

func init() {
	validateCmd.AddCommand(validateConnectionsCmd)
}

func validateConnections(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContext)
	exitOnError("Error getting REST config for cluster", err)

	for _, item := range configs {
		message := fmt.Sprintf("Validating Connections in %q", item.clusterName)
		status.Start(message)
		fmt.Println()
		submariner := getSubmarinerResource(item.config)

		if submariner == nil {
			status.QueueWarningMessage(submMissingMessage)
			status.End(cli.Success)
			continue
		}

		gateways := submariner.Status.Gateways
		if gateways == nil {
			message = "There are no gateways detected in cluster"
			status.QueueFailureMessage(message)
			status.End(cli.Failure)
			continue
		}

		allConnectionsEstablished := true
		for _, gateway := range *gateways {
			for _, connection := range gateway.Connections {
				if connection.Status == submv1.Connecting {
					message = fmt.Sprintf("Connection to cluster %q is still trying to connect", connection.Endpoint.ClusterID)
					status.QueueFailureMessage(message)
					allConnectionsEstablished = false
				} else if connection.Status == submv1.ConnectionError {
					message = fmt.Sprintf("Connection to cluster %q is not established", connection.Endpoint.ClusterID)
					status.QueueFailureMessage(message)
					allConnectionsEstablished = false
				}
			}
		}

		if !allConnectionsEstablished {
			status.End(cli.Failure)
			continue
		}

		message = "All Gateways connections are established"
		status.QueueSuccessMessage(message)
		status.End(cli.Success)
	}
}
