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
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils/restconfig"
)

var showAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Show information related to a submariner cluster",
	Long: `This command shows information related to a submariner cluster:
 networks, endpoints, gateways, connections and component versions.`,
	PreRunE: checkVersionMismatch,
	Run:     showAll,
}

func init() {
	showCmd.AddCommand(showAllCmd)
}

func showAll(cmd *cobra.Command, args []string) {
	configs, err := restconfig.ForClusters(kubeConfig, kubeContexts)
	utils.ExitOnError("Error getting REST config for cluster", err)

	for _, item := range configs {
		fmt.Println()
		fmt.Printf("Showing information for cluster %q:\n", item.ClusterName)

		fmt.Println("Showing Network details")
		showNetworkSingleCluster(item.Config)
		fmt.Println("")

		submariner := getSubmarinerResource(item.Config)

		if submariner == nil {
			fmt.Println(SubmMissingMessage)
			continue
		}

		fmt.Println("\nShowing Endpoint details")
		showEndpointsFor(submariner)
		fmt.Println("\nShowing Connection details")
		showConnectionsFor(submariner)
		fmt.Println("\nShowing Gateway details")
		showGatewaysFor(submariner)
		fmt.Println("\nShowing version details")
		showVersionsFor(item.Config, submariner)
	}
}
