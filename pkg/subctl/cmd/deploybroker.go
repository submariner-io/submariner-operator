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

	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
	lighthouse "github.com/submariner-io/submariner-operator/pkg/subctl/lighthouse/deploy"
)

var (
	enableDataplane              bool
	disableDataplane             bool
	ipsecSubmFile                string
	serviceDiscovery             bool
	serviceDiscoveryImageRepo    string
	serviceDiscoveryImageVersion string
)

func init() {
	deployBroker.PersistentFlags().BoolVar(&enableDataplane, "dataplane", false,
		"Install the Submariner dataplane on the broker")
	deployBroker.PersistentFlags().BoolVar(&disableDataplane, "no-dataplane", true,
		"Don't install the Submariner dataplane on the broker (default)")
	deployBroker.PersistentFlags().BoolVar(&serviceDiscovery, "service-discovery", false,
		"Enable Multi Cluster Service Discovery")
	deployBroker.PersistentFlags().StringVar(&serviceDiscoveryImageRepo, "service-discovery-repo", "",
		"Service Discovery Image repository")
	deployBroker.PersistentFlags().StringVar(&serviceDiscoveryImageVersion, "service-discovery-version", "",
		"Service Discovery Image version")
	err := deployBroker.PersistentFlags().MarkHidden("no-dataplane")
	// An error here indicates a programming error (the argument isn’t declared), panic
	panicOnError(err)

	deployBroker.PersistentFlags().StringVar(&ipsecSubmFile, "ipsec-psk-from", "",
		"Import IPSEC PSK from existing submariner broker file, like broker-info.subm")

	addJoinFlags(deployBroker)
	rootCmd.AddCommand(deployBroker)
}

const brokerDetailsFilename = "broker-info.subm"

var deployBroker = &cobra.Command{
	Use:   "deploy-broker",
	Short: "set the broker up",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := getRestConfig()
		exitOnError("The provided kubeconfig is invalid", err)

		status := cli.NewStatus()

		status.Start("Deploying broker")
		err = broker.Ensure(config)
		status.End(err == nil)
		exitOnError("Error deploying the broker", err)

		status.Start(fmt.Sprintf("Creating %s file", brokerDetailsFilename))

		// If deploy-broker is retried we will attempt to re-use the existing IPSEC PSK secret
		if ipsecSubmFile == "" {
			if _, err := datafile.NewFromFile(brokerDetailsFilename); err == nil {
				ipsecSubmFile = brokerDetailsFilename
				status.QueueSuccessMessage(fmt.Sprintf("Reusing IPSEC PSK from existing %s", brokerDetailsFilename))
			} else {
				status.QueueSuccessMessage(fmt.Sprintf("A new IPSEC PSK will be generated for %s", brokerDetailsFilename))
			}
		}

		subctlData, err := datafile.NewFromCluster(config, broker.SubmarinerBrokerNamespace, ipsecSubmFile)
		exitOnError("Error retrieving preparing the subm data file", err)

		newFilename, err := datafile.BackupIfExists(brokerDetailsFilename)
		exitOnError("Error backing up the brokerfile", err)

		if newFilename != "" {
			status.QueueSuccessMessage(fmt.Sprintf("Backed up previous %s to %s", brokerDetailsFilename, newFilename))
		}

		subctlData.ServiceDiscovery = serviceDiscovery

		err = subctlData.WriteToFile(brokerDetailsFilename)
		status.End(err == nil)
		exitOnError("Error writing the broker information", err)

		if serviceDiscovery {
			status.Start("Deploying Service Discovery controller")
			err = lighthouse.Ensure(config, serviceDiscoveryImageRepo, serviceDiscoveryImageVersion)
			status.End(err == nil)
			exitOnError("Failed to deploy Service Discovery controller", err)
		}

		if enableDataplane {
			joinSubmarinerCluster(subctlData)
		}
	},
}
