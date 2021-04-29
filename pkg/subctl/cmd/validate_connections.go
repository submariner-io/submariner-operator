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

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	"k8s.io/client-go/rest"
)

var validateConnectionsCmd = &cobra.Command{
	Use:   "connections",
	Short: "Check the Gateway connections",
	Long:  "This command checks that the Gateway connections to other clusters are all established",
	Run:   validateConnections,
}

func init() {
	validateCmd.AddCommand(validateConnectionsCmd)
}

func validateConnections(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContexts)
	exitOnError("Error getting REST config for cluster", err)

	validationStatus := true

	for _, item := range configs {
		status.Start(fmt.Sprintf("Retrieving Submariner resource from %q", item.clusterName))
		submariner := getSubmarinerResource(item.config)
		if submariner == nil {
			status.QueueWarningMessage(submMissingMessage)
			status.End(cli.Success)
			continue
		}
		status.End(cli.Success)
		validationStatus = validationStatus && validateConnectionsInCluster(item.config, item.clusterName)
	}
	if !validationStatus {
		os.Exit(1)
	}
}

func validateConnectionsInCluster(config *rest.Config, clusterName string) bool {
	message := fmt.Sprintf("Checking Gateway connections in cluster %q", clusterName)
	status.Start(message)

	gateways := getGatewaysResource(config)
	if gateways == nil {
		message = "There are no gateways detected"
		status.QueueWarningMessage(message)
		status.End(cli.Failure)
		return false
	}

	allConnectionsEstablished := true
	for _, gateway := range gateways.Items {
		if gateway.Status.HAStatus != submv1.HAStatusActive {
			continue
		}

		if len(gateway.Status.Connections) == 0 {
			message = "There are no active connections"
			status.QueueFailureMessage(message)
			status.End(cli.Failure)
			return false
		}

		for _, connection := range gateway.Status.Connections {
			if connection.Status == submv1.Connecting {
				message = fmt.Sprintf("Connection to cluster %q is in progress", connection.Endpoint.ClusterID)
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
		return false
	}

	message = "All connections are established"
	status.QueueSuccessMessage(message)
	status.End(cli.Success)
	return true
}
