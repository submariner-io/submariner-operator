/*
© 2019 Red Hat, Inc. and others.

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
	"github.com/spf13/cobra"

	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
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
	addKubeconfigFlag(infoCmd)
	rootCmd.AddCommand(infoCmd)
}

func clusterInfo(cmd *cobra.Command, args []string) {

	dynClient, clientSet, err := getClients()
	panicOnError(err)

	clusterNetwork, err := network.Discover(dynClient, clientSet)
	exitOnError("There was an error discovering network details for this cluster", err)

	clusterNetwork.Show()

}
