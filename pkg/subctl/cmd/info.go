package cmd

import (
	"fmt"

	"github.com/submariner-io/submariner-operator/pkg/discovery/network"

	"github.com/spf13/cobra"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get information on your cluster related to submariner",
	Long: `This command shows the status of submariner in your cluster,
and the relevant network details from your cluster.`,
	Run: clusterInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func clusterInfo(cmd *cobra.Command, args []string) {

	dynClient, clientSet, err := getClients()
	panicOnError(err)

	clusterNetwork, err := network.Discover(dynClient, clientSet)
	if err != nil {
		fmt.Printf("There was an error discovering network details for this cluster: %s", err)
	}

	fmt.Printf("cluster network info: %v\n", clusterNetwork)
}
