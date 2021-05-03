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
	"os"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
)

var validateAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Run all diagnostic checks (except those requiring two kubecontexts)",
	Long:  "This command runs all diagnostic checks (except those requiring two kubecontexts) and reports any issues",
	Run:   validateAll,
}

func init() {
	validateCmd.AddCommand(validateAllCmd)
}

func validateAll(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContexts)
	exitOnError("Error getting REST config for cluster", err)

	validationStatus := true

	for _, item := range configs {
		validationStatus = validationStatus && validateK8sVersionInCluster(item.config, item.clusterName)
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

		validationStatus = validationStatus && validateCNIInCluster(item.config, item.clusterName, submariner)
		fmt.Println()
		validationStatus = validationStatus && validateConnectionsInCluster(item.config, item.clusterName)
		fmt.Println()
		validationStatus = validationStatus && checkPods(item, submariner, OperatorNamespace)
		fmt.Println()
		validationStatus = validationStatus && checkOverlappingCIDRs(item, submariner)
		fmt.Println()
		validationStatus = validationStatus && validateKubeProxyModeInCluster(item.config, item.clusterName)
		fmt.Println()
		validationStatus = validationStatus && validateFirewallMetricsConfigWithinCluster(item.config, item.clusterName)
		fmt.Println()
		validationStatus = validationStatus && validateVxLANConfigWithinCluster(item.config, item.clusterName, submariner)
		fmt.Println()
		fmt.Printf("Skipping tunnel firewall check as it requires two kubeconfigs." +
			" Please run \"subctl diagnose firewall tunnel\" command manually.\n")
		fmt.Println()
	}

	if !validationStatus {
		os.Exit(1)
	}
}
