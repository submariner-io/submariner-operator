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
)

func validateK8sConfig(cmd *cobra.Command, args []string) {
	fmt.Println("\nValidating CNI Configuration")
	fmt.Println("\n----------------------------")
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContext)
	exitOnError("Error getting REST config for cluster", err)

	for _, item := range configs {
		fmt.Println()
		submariner := getSubmarinerResource(item.config)

		if submariner == nil {
			fmt.Println(submMissingMessage)
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
			fmt.Printf("The detected CNI network plugin (%q) is not supported by Submariner."+
				" Supported network plugins: %v\n", submariner.Status.NetworkPlugin, supportedNetworkPlugins)
			continue
		}

		fmt.Printf("The detected CNI network plugin (%q) is supported by Submariner.\n",
			submariner.Status.NetworkPlugin)
	}
}
