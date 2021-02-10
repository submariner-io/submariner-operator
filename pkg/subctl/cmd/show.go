/*
© 2019 Red Hat, Inc. and others.

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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
)

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show information about submariner",
	Long:  `This command shows information about some aspect of the submariner deployment in a cluster.`,
}

const submMissingMessage = "Submariner is not installed"

type restConfig struct {
	config      *rest.Config
	clusterName string
}

func init() {
	addKubeconfigFlag(showCmd)
	rootCmd.AddCommand(showCmd)
}

func getMultipleRestConfigs(kubeConfigPath, kubeContext string) ([]restConfig, error) {
	var restConfigs []restConfig

	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
	rules.ExplicitPath = kubeConfigPath

	if kubeContext != "" {
		overrides.CurrentContext = kubeContext
	}

	if kubeConfigPath != "" || kubeContext != "" {
		config, err := getClientConfigAndClusterName(rules, overrides)
		if err != nil {
			return nil, err
		}
		restConfigs = append(restConfigs, config)
		return restConfigs, nil
	}

	for _, item := range rules.Precedence {
		if item != "" {
			rules.ExplicitPath = item
			config, err := getClientConfigAndClusterName(rules, overrides)
			if err != nil {
				return nil, err
			}

			restConfigs = append(restConfigs, config)
		}
	}

	return restConfigs, nil
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
		return restConfig{}, fmt.Errorf("Could not obtain the cluster name from kube config: %#v", raw)
	}

	return restConfig{config: clientConfig, clusterName: *clusterName}, nil
}

func getSubmarinerResource(config *rest.Config) *v1alpha1.Submariner {
	submarinerClient, err := submarinerclientset.NewForConfig(config)
	exitOnError("Unable to get the Submariner client", err)

	submariner, err := submarinerClient.SubmarinerV1alpha1().Submariners(OperatorNamespace).Get(submarinercr.SubmarinerName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		exitOnError("Error obtaining the Submariner resource", err)
	}

	return submariner
}
