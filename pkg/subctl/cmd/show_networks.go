package cmd

import (
	"github.com/spf13/cobra"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
)

// showNetworksCmd represents the show networks command
var showNetworksCmd = &cobra.Command{
	Use:   "networks",
	Short: "Get information on your cluster related to submariner",
	Long: `This command shows the status of submariner in your cluster,
and the relevant network details from your cluster.`,
	Run: showNetwork,
}

func init() {
	showCmd.AddCommand(showNetworksCmd)
}

func showNetwork(cmd *cobra.Command, args []string) {
	config, err := getRestConfig(kubeConfig, kubeContext)
	exitOnError("Error getting REST config for cluster", err)
	dynClient, clientSet, err := getClients(config)
	exitOnError("Error creating clients for cluster", err)

	submarinerClient, err := submarinerclientset.NewForConfig(config)
	exitOnError("Unable to get the Submariner client", err)

	clusterNetwork, err := network.Discover(dynClient, clientSet, submarinerClient, OperatorNamespace)
	exitOnError("There was an error discovering network details for this cluster", err)

	clusterNetwork.Show()
}
