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
	"fmt"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
)

func init() {
	diagnoseCmd.AddCommand(&cobra.Command{
		Use:   "all",
		Short: "Run all diagnostic checks (except those requiring two kubecontexts)",
		Long:  "This command runs all diagnostic checks (except those requiring two kubecontexts) and reports any issues",
		Run: func(command *cobra.Command, args []string) {
			cmd.ExecuteMultiCluster(restConfigProducer, diagnoseAll)
		},
	})
}

func diagnoseAll(cluster *cmd.Cluster) bool {
	success := checkK8sVersion(cluster)

	fmt.Println()

	status := cli.NewStatus()
	if cluster.Submariner == nil {
		status.Start(cmd.SubmMissingMessage)
		status.EndWith(cli.Warning)

		return success
	}

	success = checkCNIConfig(cluster) && success

	fmt.Println()

	success = checkConnections(cluster) && success

	fmt.Println()

	success = checkDeployments(cluster) && success

	fmt.Println()

	success = checkKubeProxyMode(cluster) && success

	fmt.Println()

	success = checkFirewallMetricsConfig(cluster) && success

	fmt.Println()

	success = checkVxLANConfig(cluster) && success

	fmt.Println()

	success = checkGlobalnet(cluster) && success

	fmt.Println()

	fmt.Printf("Skipping inter-cluster firewall check as it requires two kubeconfigs." +
		" Please run \"subctl diagnose firewall inter-cluster\" command manually.\n")

	return success
}
