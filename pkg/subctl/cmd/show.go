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

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show information about submariner",
	Long:  `This command shows information about some aspect of the submariner deployment in a cluster.`,
}

const SubmMissingMessage = "Submariner is not installed"

type restConfig struct {
	config      *rest.Config
	clusterName string
}

func init() {
	AddKubeContextFlag(showCmd)
	rootCmd.AddCommand(showCmd)
}

func getClientConfigAndClusterName(rules *clientcmd.ClientConfigLoadingRules, overrides *clientcmd.ConfigOverrides) (restConfig, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
	clientConfig, err := config.ClientConfig()
	if err != nil {
		return restConfig{}, err
	}

	raw, err := config.RawConfig()
	if err != nil {
		return restConfig{}, err
	}

	clusterName := getClusterNameFromContext(raw, overrides.CurrentContext)

	if clusterName == nil {
		return restConfig{}, fmt.Errorf("could not obtain the cluster name from kube config: %#v", raw)
	}

	return restConfig{config: clientConfig, clusterName: *clusterName}, nil
}
