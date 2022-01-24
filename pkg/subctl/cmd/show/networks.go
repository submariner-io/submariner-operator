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
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/exit"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
)

func init() {
	showCmd.AddCommand(&cobra.Command{
		Use:   "networks",
		Short: "Get information on your cluster related to submariner",
		Long: `This command shows the status of submariner in your cluster,
		      and the relevant network details from your cluster.`,
		PreRunE: restConfigProducer.CheckVersionMismatch,
		Run: func(command *cobra.Command, args []string) {
			cmd.ExecuteMultiCluster(restConfigProducer, showNetwork)
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
		exit.OnErrorWithMessage(err, "Unable to get the Submariner client")

		clusterNetwork, err = network.Discover(cluster.DynClient, cluster.KubeClient, submarinerClient, cmd.OperatorNamespace)
		exit.OnErrorWithMessage(err, "There was an error discovering network details for this cluster")
	}

	if clusterNetwork != nil {
		fmt.Println(msg)
	}

	clusterNetwork.Show()
	status.EndWith(cli.Success)

	return true
}
