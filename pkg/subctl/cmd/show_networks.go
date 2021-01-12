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
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"k8s.io/client-go/rest"
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
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContext)
	exitOnError("Error getting REST config for cluster", err)

	for _, item := range configs {
		fmt.Println()
		fmt.Printf("Showing network details for cluster %q:\n", item.clusterName)
		showNetworkSingleCluster(item.config)
	}
}

func showNetworkSingleCluster(config *rest.Config) {
	dynClient, clientSet, err := getClients(config)
	exitOnError("Error creating clients for cluster", err)

	submarinerClient, err := submarinerclientset.NewForConfig(config)
	exitOnError("Unable to get the Submariner client", err)

	clusterNetwork, err := network.Discover(dynClient, clientSet, submarinerClient, OperatorNamespace)
	exitOnError("There was an error discovering network details for this cluster", err)

	clusterNetwork.Show()
}
