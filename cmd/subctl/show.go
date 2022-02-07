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

package subctl

import (
	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/cmd/subctl/execute"
	"github.com/submariner-io/submariner-operator/internal/show"
)

var (
	// showCmd represents the show command.
	showCmd = &cobra.Command{
		Use:   "show",
		Short: "Show information about submariner",
		Long:  `This command shows information about some aspect of the submariner deployment in a cluster.`,
	}
	connectionsCmd = &cobra.Command{
		Use:     "connections",
		Short:   "Show cluster connectivity information",
		Long:    `This command shows information about submariner endpoint connections with other clusters.`,
		PreRunE: restConfigProducer.CheckVersionMismatch,
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, show.Connections)
		},
	}
	endpointsCmd = &cobra.Command{
		Use:     "endpoints",
		Short:   "Show submariner endpoint information",
		Long:    `This command shows information about submariner endpoints in a cluster.`,
		PreRunE: restConfigProducer.CheckVersionMismatch,
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, show.Endpoints)
		},
	}
	gatewaysCmd = &cobra.Command{
		Use:     "gateways",
		Short:   "Show submariner gateway summary information",
		Long:    `This command shows summary information about the submariner gateways in a cluster.`,
		PreRunE: restConfigProducer.CheckVersionMismatch,
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, show.Gateways)
		},
	}
	networkCmd = &cobra.Command{
		Use:   "networks",
		Short: "Get information on your cluster related to submariner",
		Long: `This command shows the status of submariner in your cluster,
		      and the relevant network details from your cluster.`,
		PreRunE: restConfigProducer.CheckVersionMismatch,
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, show.Network)
		},
	}
	versionsCmd = &cobra.Command{
		Use:     "versions",
		Short:   "Shows submariner component versions",
		Long:    `This command shows the versions of the submariner components in the cluster.`,
		PreRunE: restConfigProducer.CheckVersionMismatch,
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, show.Versions)
		},
	}
	allCmd = &cobra.Command{
		Use:   "all",
		Short: "Show information related to a submariner cluster",
		Long: `This command shows information related to a submariner cluster:
		      networks, endpoints, gateways, connections and component versions.`,
		PreRunE: restConfigProducer.CheckVersionMismatch,
		Run: func(command *cobra.Command, args []string) {
			execute.OnMultiCluster(restConfigProducer, show.All)
		},
	}
)

func init() {
	restConfigProducer.AddKubeConfigFlag(showCmd)
	rootCmd.AddCommand(showCmd)
	showCmd.AddCommand(connectionsCmd)
	showCmd.AddCommand(endpointsCmd)
	showCmd.AddCommand(gatewaysCmd)
	showCmd.AddCommand(networkCmd)
	showCmd.AddCommand(versionsCmd)
	showCmd.AddCommand(allCmd)
}
