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
		fmt.Printf("Showing information for cluster %q:\n", item.context)

		showNetworkSingleCluster(item.config)
		fmt.Println()
		showEndpointsFromConfig(item.config)
		fmt.Println()
		showConnectionsFromConfig(item.config)
		fmt.Println()
		showGatewaysFromConfig(item.config)
		fmt.Println()
		showVersionsFromConfig(item.config)
	}
}
