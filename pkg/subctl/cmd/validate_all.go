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
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
)

var validateAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Validate the Submariner deployment and report any issues",
	Long:  "This command validates the Submariner deployment and reports any issues",
	Run:   validateAll,
}

func init() {
	validateCmd.AddCommand(validateAllCmd)
}

func validateAll(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContexts)
	exitOnError("Error getting REST config for cluster", err)

	for _, item := range configs {
		validateK8sVersionInCluster(item.config, item.clusterName)
		fmt.Println()

		status.Start(fmt.Sprintf("Retrieving Submariner resource from %q", item.clusterName))
		submariner := getSubmarinerResource(item.config)
		if submariner == nil {
			status.QueueWarningMessage(submMissingMessage)
			status.End(cli.Success)
			fmt.Println()
			continue
		}
		status.End(cli.Success)
		fmt.Println()

		validateCNIInCluster(item.clusterName, submariner)
		fmt.Println()
		validateConnectionsInCluster(item.config, item.clusterName)
		fmt.Println()
		checkPods(item, submariner, OperatorNamespace)
		fmt.Println()
		checkOverlappingCIDRs(item, submariner)
		fmt.Println()
		validateKubeProxyModeInCluster(item.config, item.clusterName)
		fmt.Println()
		validateFirewallMetricsConfigWithinCluster(item.config, item.clusterName)
		fmt.Println()
		validateVxLANConfigWithinCluster(item.config, item.clusterName, submariner)
		fmt.Println()
		fmt.Printf("Skipping tunnel firewall validation as it requires two kubeconfigs." +
			" Please run \"subctl validate firewall tunnel\" command manually.\n")
		fmt.Println()
	}
}
