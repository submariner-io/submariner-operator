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
	showNetwork(cmd, args)
	fmt.Println()
	showEndpoints(cmd, args)
	fmt.Println()
	showConnections(cmd, args)
	fmt.Println()
	showGateways(cmd, args)
	fmt.Println()
	showVersions(cmd, args)
}
