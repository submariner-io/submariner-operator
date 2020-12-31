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
		showVersionsFor(item.config, submariner)
	}
}
