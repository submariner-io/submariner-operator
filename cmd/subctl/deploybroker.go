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

package subctl

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/submariner-io/admiral/pkg/stringset"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/component"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/deploy"
)

var (
	deployflags       deploy.BrokerOptions
	ipsecSubmFile     string
	defaultComponents = []string{component.ServiceDiscovery, component.Connectivity}
)

// deployBroker represents the deployBroker command.
var deployBroker = &cobra.Command{
	Use:   "deploy-broker",
	Short: "Deploys the broker",
	Run: func(cmd *cobra.Command, args []string) {
		status := cli.NewReporter()

		config, err := restConfigProducer.ForCluster()
		exit.OnError(status.Error(err, "Error creating REST config"))

		clientProducer, err := client.NewProducerFromRestConfig(config.Config)
		exit.OnError(status.Error(err, "Error creating client producer"))

		err = deploy.Broker(&deployflags, clientProducer, status)
		exit.OnError(err)

		err = broker.WriteInfoToFile(config.Config, deployflags.BrokerNamespace, ipsecSubmFile,
			stringset.New(deployflags.BrokerSpec.Components...), deployflags.BrokerSpec.DefaultCustomDomains, status)
		exit.OnError(err)
	},
}

func init() {
	addDeployBrokerFlags()
	restConfigProducer.AddKubeContextFlag(deployBroker)
	rootCmd.AddCommand(deployBroker)
}

func addDeployBrokerFlags() {
	deployBroker.PersistentFlags().BoolVar(&deployflags.BrokerSpec.GlobalnetEnabled, "globalnet", false,
		"enable support for Overlapping CIDRs in connecting clusters (default disabled)")
	deployBroker.PersistentFlags().StringVar(&deployflags.BrokerSpec.GlobalnetCIDRRange, "globalnet-cidr-range",
		broker.DefaultGlobalnetCIDR, "GlobalCIDR supernet range for allocating GlobalCIDRs to each cluster")
	deployBroker.PersistentFlags().UintVar(&deployflags.BrokerSpec.DefaultGlobalnetClusterSize, "globalnet-cluster-size",
		broker.DefaultGlobalnetClusterSize, "default cluster size for GlobalCIDR allocated to each cluster (amount of global IPs)")

	deployBroker.PersistentFlags().StringVar(&ipsecSubmFile, "ipsec-psk-from", "",
		"import IPsec PSK from existing submariner broker file, like broker-info.subm")

	deployBroker.PersistentFlags().StringSliceVar(&deployflags.BrokerSpec.DefaultCustomDomains, "custom-domains", nil,
		"list of domains to use for multicluster service discovery")

	deployBroker.PersistentFlags().StringSliceVar(&deployflags.BrokerSpec.Components, "components", defaultComponents,
		fmt.Sprintf("The components to be installed - any of %s", strings.Join(deploy.ValidComponents, ",")))

	deployBroker.PersistentFlags().StringVar(&deployflags.Repository, "repository", "", "image repository")
	deployBroker.PersistentFlags().StringVar(&deployflags.ImageVersion, "version", "", "image version")

	deployBroker.PersistentFlags().BoolVar(&deployflags.OperatorDebug, "operator-debug", false, "enable operator debugging (verbose logging)")
	deployBroker.PersistentFlags().StringVar(&deployflags.BrokerNamespace, "broker-namespace", constants.DefaultBrokerNamespace,
		"namespace for broker")
}
