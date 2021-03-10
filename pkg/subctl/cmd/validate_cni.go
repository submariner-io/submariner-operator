/*
© 2021 Red Hat, Inc. and others.

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

var supportedNetworkPlugins = []string{"generic", "canal-flannel", "weave-net", "OpenShiftSDN", "OVNKubernetes"}

var validateCniCmd = &cobra.Command{
	Use:   "cni",
	Short: "Validate if Submariner supports the CNI network plugin.",
	Long: "This command validates if the detected CNI network plugin is supported by Submariner or not.",
	Run: validateCniConfig,
}

func init() {
	validateCmd.AddCommand(validateCniCmd)
}

func validateCniConfig(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContext)
	exitOnError("Error getting REST config for cluster", err)

	for _, item := range configs {
		message := fmt.Sprintf("Validating Submariner support for CNI network plugin in %q", item.clusterName)
		status.Start(message)
		fmt.Println()
		submariner := getSubmarinerResource(item.config)

		if submariner == nil {
			status.QueueWarningMessage(submMissingMessage)
			status.End(cli.Success)
			continue
		}

		isSupportedPlugin := false
		for _, np := range supportedNetworkPlugins {
			if submariner.Status.NetworkPlugin == np {
				isSupportedPlugin = true
				break
			}
		}

		if !isSupportedPlugin {
			message = fmt.Sprintf("The detected CNI network plugin (%q) is not supported by Submariner."+
				" Supported network plugins: %v\n", submariner.Status.NetworkPlugin, supportedNetworkPlugins)
			status.QueueFailureMessage(message)
			status.End(cli.Failure)
			continue
		}

		message = fmt.Sprintf("The detected CNI network plugin (%q) is supported by Submariner.\n",
			submariner.Status.NetworkPlugin)
		status.QueueSuccessMessage(message)
		status.End(cli.Success)
	}
}
