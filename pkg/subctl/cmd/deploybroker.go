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
	v1 "k8s.io/api/core/v1"

	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"

	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
)

var (
	ipsecSubmFile               string
	globalnetEnable             bool
	globalnetCidrRange          string
	defaultGlobalnetClusterSize uint
	serviceDiscovery            bool
	GlobalCIDRConfigMap         *v1.ConfigMap
)

func init() {
	deployBroker.PersistentFlags().BoolVar(&globalnetEnable, "globalnet", false,
		"Enable support for Overlapping CIDRs in connecting clusters (default disabled)")
	deployBroker.PersistentFlags().StringVar(&globalnetCidrRange, "globalnet-cidr-range", "169.254.0.0/16",
		"Global CIDR supernet range for allocating GlobalCIDRs to each cluster")
	deployBroker.PersistentFlags().UintVar(&defaultGlobalnetClusterSize, "globalnet-cluster-size", 8192,
		"Default cluster size for GlobalCIDR allocated to each cluster (amount of global IPs)")

	deployBroker.PersistentFlags().StringVar(&ipsecSubmFile, "ipsec-psk-from", "",
		"Import IPsec PSK from existing submariner broker file, like broker-info.subm")

	deployBroker.PersistentFlags().BoolVar(&serviceDiscovery, "service-discovery", false,
		"Enable Multi Cluster Service Discovery")

	addKubeconfigFlag(deployBroker)
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

		status := cli.NewStatus()
		status.Start("Deploying broker")
		err = broker.Ensure(config)
		status.End(cli.CheckForError(err))
		exitOnError("Error deploying the broker", err)

		status.Start(fmt.Sprintf("Creating %s file", brokerDetailsFilename))

		// If deploy-broker is retried we will attempt to re-use the existing IPsec PSK secret
		if ipsecSubmFile == "" {
			if _, err := datafile.NewFromFile(brokerDetailsFilename); err == nil {
				ipsecSubmFile = brokerDetailsFilename
				status.QueueWarningMessage(fmt.Sprintf("Reusing IPsec PSK from existing %s", brokerDetailsFilename))
			} else {
				status.QueueSuccessMessage(fmt.Sprintf("A new IPsec PSK will be generated for %s", brokerDetailsFilename))
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

		exitOnError("Error setting up service discovery information", err)

		err = broker.CreateGlobalnetConfigMap(config, globalnetEnable, globalnetCidrRange,
			defaultGlobalnetClusterSize, broker.SubmarinerBrokerNamespace)
		exitOnError("Error creating globalCIDR configmap on Broker", err)

		err = subctlData.WriteToFile(brokerDetailsFilename)
		status.End(cli.CheckForError(err))
		exitOnError("Error writing the broker information", err)

	},
}

func isValidGlobalnetConfig() (bool, error) {
	var err error
	if !globalnetEnable {
		return true, nil
	}
	defaultGlobalnetClusterSize, err = globalnet.GetValidClusterSize(globalnetCidrRange, defaultGlobalnetClusterSize)
	if err != nil || defaultGlobalnetClusterSize == 0 {
		return false, err
	}
	return true, err
}
