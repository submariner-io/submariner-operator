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
	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/diagnose"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
)

func init() {
	diagnoseCmd.AddCommand(&cobra.Command{
		Use:   "deployment",
		Short: "Check the Submariner deployment",
		Long:  "This command checks that the Submariner components are properly deployed and running with no overlapping CIDRs.",
		Run: func(command *cobra.Command, args []string) {
			cmd.ExecuteMultiCluster(restConfigProducer, checkDeployments)
		},
	})
}

func checkDeployments(c *cmd.Cluster) bool {
	return diagnose.Deployments(clusterInfoFrom(c), cli.NewStatus())
}
