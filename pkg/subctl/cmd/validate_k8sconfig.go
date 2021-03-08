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
				// SGM: Check the networkPlugin in submariner route-agent matches with route-agent.
				isSupportedPlugin = true
				break
			}
		}

		if !isSupportedPlugin {
			fmt.Println("Submariner is not validated against the cluster's CNI (Network Plugin):",
				submariner.Status.NetworkPlugin)
			fmt.Println("Supported Network Plugins:", supportedNetworkPlugins)
			continue
		}

		routeAgent := getK8sDaemonSet(item.config, "submariner-routeagent")
		if routeAgent == nil {
			fmt.Println("Submariner route agent not found")
			continue
		}

		for _, env := range routeAgent.Spec.Template.Spec.Containers[0].Env {
			if env.Name == "SUBMARINER_NETWORKPLUGIN" && env.Value != submariner.Status.NetworkPlugin {
				fmt.Printf("SUBMARINER_NETWORKPLUGIN configured in route agent (%q) does"+
					" not match with the one in Submariner's CR (%q) \n", env.Value, submariner.Status.NetworkPlugin)
				continue
			}
		}
		fmt.Printf("Detected Network Plugin (%q) is supported by Submariner.\n", submariner.Status.NetworkPlugin)
	}
}
