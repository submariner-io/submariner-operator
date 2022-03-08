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
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
)

func init() {
	showCmd.AddCommand(&cobra.Command{
		Use:   "all",
		Short: "Show information related to a submariner cluster",
		Long: `This command shows information related to a submariner cluster:
		      networks, endpoints, gateways, connections and component versions.`,
		PreRunE: restConfigProducer.CheckVersionMismatch,
		Run: func(command *cobra.Command, args []string) {
			cmd.ExecuteMultiCluster(restConfigProducer, showAll)
		},
	})
}

func showAll(cluster *cmd.Cluster) bool {
	status := cli.NewStatus()

	success := showBrokers(cluster)

	fmt.Println()

	if cluster.Submariner == nil {
		success = getVersions(cluster) && success

		fmt.Println()
		status.Start(cmd.SubmMissingMessage)
		status.EndWith(cli.Warning)

		return success
	}

	success = showConnections(cluster) && success

	fmt.Println()

	success = showEndpoints(cluster) && success

	fmt.Println()

	success = showGateways(cluster) && success

	fmt.Println()

	success = showNetwork(cluster) && success

	fmt.Println()

	success = showVersions(cluster) && success

	return success
}
