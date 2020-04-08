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
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"

	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
	lighthouse "github.com/submariner-io/submariner-operator/pkg/subctl/lighthouse/deploy"
)

var (
	enableDataplane             bool
	disableDataplane            bool
	ipsecSubmFile               string
	globalnetEnable             bool
	globalnetCidrRange          string
	defaultGlobalnetClusterSize uint
)

func init() {
	deployBroker.PersistentFlags().BoolVar(&enableDataplane, "dataplane", false,
		"Install the Submariner dataplane on the broker")
	deployBroker.PersistentFlags().BoolVar(&disableDataplane, "no-dataplane", true,
		"Don't install the Submariner dataplane on the broker (default)")
	// TODO (skitt) make this generic for potentially multiple plugins (see below too)
	lighthouse.AddFlags(deployBroker, "service-discovery")

	deployBroker.PersistentFlags().BoolVar(&globalnetEnable, "globalnet", false,
		"Enable support for Overlapping CIDRs in connecting clusters (default disabled)")
	deployBroker.PersistentFlags().StringVar(&globalnetCidrRange, "globalnet-cidr-range", "169.254.0.0/16",
		"Global CIDR supernet range for allocating GlobalCIDRs to each cluster")
	deployBroker.PersistentFlags().UintVar(&defaultGlobalnetClusterSize, "globalnet-cluster-size", 8192,
		"Default cluster size for GlobalCIDR allocated to each cluster (amount of global IPs)")
	err := deployBroker.PersistentFlags().MarkHidden("no-dataplane")
	// An error here indicates a programming error (the argument isn’t declared), panic
	panicOnError(err)

	deployBroker.PersistentFlags().StringVar(&ipsecSubmFile, "ipsec-psk-from", "",
		"Import IPSEC PSK from existing submariner broker file, like broker-info.subm")

	addKubeconfigFlag(deployBroker)
	addJoinFlags(deployBroker)
	rootCmd.AddCommand(deployBroker)
}

const brokerDetailsFilename = "broker-info.subm"

var deployBroker = &cobra.Command{
	Use:   "deploy-broker",
	Short: "set the broker up",
	Run: func(cmd *cobra.Command, args []string) {

		if valid, err := isValidGlobalnetConfig(); !valid {
			exitOnError("Invalid GlobalCidr configuration", err)
		}

		config, err := getRestConfig(kubeConfig, kubeContext)
		exitOnError("The provided kubeconfig is invalid", err)

		err = lighthouse.Validate()
		exitOnError("Invalid configuration", err)

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

		err = lighthouse.FillSubctlData(subctlData)
		exitOnError("Error setting up service discovery information", err)

		if globalnetEnable {
			subctlData.GlobalnetCidrRange = globalnetCidrRange
			subctlData.GlobalnetClusterSize = defaultGlobalnetClusterSize
		}

		err = subctlData.WriteToFile(brokerDetailsFilename)
		status.End(err == nil)
		exitOnError("Error writing the broker information", err)

		err = lighthouse.HandleCommand(status, config, true, kubeConfig, kubeContext)
		exitOnError("Error setting up service discovery", err)

		if enableDataplane {
			joinSubmarinerCluster(config, subctlData)
		}
	},
}

func isValidGlobalnetConfig() (bool, error) {
	var err error
	if !globalnetEnable {
		return true, nil
	}
	defaultGlobalnetClusterSize, err = globalnet.GetValidClusterSize(globalnetCidrRange, defaultGlobalnetClusterSize)
	if err != nil || defaultGlobalnetClusterSize <= 0 {
		return false, err
	}
	return true, err
}
