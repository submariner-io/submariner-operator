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

package diagnose

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	"github.com/submariner-io/submariner-operator/pkg/diagnose"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
)

func init() {
	diagnoseCmd.AddCommand(&cobra.Command{
		Use:   "cni",
		Short: "Check the CNI network plugin",
		Long:  "This command checks if the detected CNI network plugin is supported by Submariner.",
		Run: func(command *cobra.Command, args []string) {
			cmd.ExecuteMultiCluster(restConfigProducer, checkCNIConfig)
		},
	})
}

func checkCNIConfig(c *cmd.Cluster) bool {
	return diagnose.CNIConfig(clusterInfoFrom(c), cli.NewStatus())
}

func clusterInfoFrom(c *cmd.Cluster) *cluster.Info {
	p, err := client.NewProducerFromRestConfig(c.Config)
	exit.OnErrorWithMessage(err, "Error creating client producer")

	i, err := cluster.NewInfo(c.Name, p, nil)
	exit.OnErrorWithMessage(err, "Error initializing client info")

	if i.Submariner == nil {
		cli.NewStatus().Warning(cmd.SubmMissingMessage)
		os.Exit(0)
	}

	return i
}
