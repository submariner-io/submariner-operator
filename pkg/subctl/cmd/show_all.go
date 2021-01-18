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
)

var showAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Show information related to a submariner cluster",
	Long: `This command shows information related to a submariner cluster:
 networks, endpoints, gateways, connections and component versions.`,
	Run: showAll,
}

func init() {
	showCmd.AddCommand(showAllCmd)
}

func showAll(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContext)
	exitOnError("Error getting REST config for cluster", err)

	for _, item := range configs {
		fmt.Println()
		fmt.Printf("Showing information for cluster %q:\n", item.clusterName)

		fmt.Println("Showing Network details")
		showNetworkSingleCluster(item.config)
		fmt.Println("")

		submariner := getSubmarinerResource(item.config)

		if submariner == nil {
			fmt.Println(submMissingMessage)
			continue
		}

		fmt.Println("\nShowing Endpoint details")
		showEndpointsFor(submariner)
		fmt.Println("\nShowing Connection details")
		showConnectionsFor(submariner)
		fmt.Println("\nShowing Gateway details")
		showGatewaysFor(submariner)
		fmt.Println("\nShowing version details")
		getVersionsFor(item.config)
	}
}
