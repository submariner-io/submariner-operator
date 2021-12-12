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
package show

import (
	"fmt"

	"github.com/spf13/cobra"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
)

func init() {
	showCmd.AddCommand(&cobra.Command{
		Use:   "networks",
		Short: "Get information on your cluster related to submariner",
		Long: `This command shows the status of submariner in your cluster,
		      and the relevant network details from your cluster.`,
		PreRunE: cmd.CheckVersionMismatch,
		Run: func(command *cobra.Command, args []string) {
			cmd.ExecuteMultiCluster(showNetwork)
		},
	})
}

func showNetwork(cluster *cmd.Cluster) bool {
	status := cli.NewStatus()
	status.Start("Showing Network details")

	var clusterNetwork *network.ClusterNetwork
	var msg string

	if cluster.Submariner != nil {
		msg = "    Discovered network details via Submariner:"
		clusterNetwork = &network.ClusterNetwork{
			PodCIDRs:      []string{cluster.Submariner.Status.ClusterCIDR},
			ServiceCIDRs:  []string{cluster.Submariner.Status.ServiceCIDR},
			NetworkPlugin: cluster.Submariner.Status.NetworkPlugin,
			GlobalCIDR:    cluster.Submariner.Status.GlobalCIDR,
		}
	} else {
		msg = "    Discovered network details"

		submarinerClient, err := submarinerclientset.NewForConfig(cluster.Config)
		utils.ExitOnError("Unable to get the Submariner client", err)

		clusterNetwork, err = network.Discover(cluster.DynClient, cluster.KubeClient, submarinerClient, cmd.OperatorNamespace)
		utils.ExitOnError("There was an error discovering network details for this cluster", err)
	}

	if clusterNetwork != nil {
		fmt.Println(msg)
	}

	clusterNetwork.Show()
	status.End(cli.Success)

	return true
}
