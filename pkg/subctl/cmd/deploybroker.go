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
	"strings"

	"github.com/spf13/cobra"
	"github.com/submariner-io/admiral/pkg/stringset"
	v1 "k8s.io/api/core/v1"

	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"
	"github.com/submariner-io/submariner-operator/pkg/subctl/components"

	submarinerv1a1 "github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/brokercr"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop"
)

var (
	ipsecSubmFile               string
	globalnetEnable             bool
	globalnetCIDRRange          string
	defaultGlobalnetClusterSize uint
	serviceDiscoveryEnabled     bool
	componentArr                []string
	GlobalCIDRConfigMap         *v1.ConfigMap
	defaultCustomDomains        []string
)

var defaultComponents = []string{components.ServiceDiscovery, components.Connectivity}
var validComponents = []string{components.ServiceDiscovery, components.Connectivity, components.Globalnet}

func init() {
	deployBroker.PersistentFlags().BoolVar(&globalnetEnable, "globalnet", false,
		"enable support for Overlapping CIDRs in connecting clusters (default disabled)")

	_ = deployBroker.PersistentFlags().MarkDeprecated("globalnet", "please use --components globalnet")

	deployBroker.PersistentFlags().StringVar(&globalnetCIDRRange, "globalnet-cidr-range", "169.254.0.0/16",
		"GlobalCIDR supernet range for allocating GlobalCIDRs to each cluster")
	deployBroker.PersistentFlags().UintVar(&defaultGlobalnetClusterSize, "globalnet-cluster-size", 8192,
		"default cluster size for GlobalCIDR allocated to each cluster (amount of global IPs)")

	deployBroker.PersistentFlags().StringVar(&ipsecSubmFile, "ipsec-psk-from", "",
		"import IPsec PSK from existing submariner broker file, like broker-info.subm")

	deployBroker.PersistentFlags().BoolVar(&serviceDiscoveryEnabled, "service-discovery", true,
		"enable multi-cluster service discovery")

	_ = deployBroker.PersistentFlags().MarkDeprecated("service-discovery", "please use --components instead")

	deployBroker.PersistentFlags().StringSliceVar(&defaultCustomDomains, "custom-domains", nil,
		"list of domains to use for multicluster service discovery")

	deployBroker.PersistentFlags().StringSliceVar(&componentArr, "components", defaultComponents,
		fmt.Sprintf("The components to be installed - any of %s",
			strings.Join(validComponents, ",")))

	deployBroker.PersistentFlags().StringVar(&repository, "repository", "", "image repository")
	deployBroker.PersistentFlags().StringVar(&imageVersion, "version", "", "image version")

	addKubeconfigFlag(deployBroker)
	rootCmd.AddCommand(deployBroker)
}

const brokerDetailsFilename = "broker-info.subm"

var deployBroker = &cobra.Command{
	Use:   "deploy-broker",
	Short: "Set the broker up",
	Run: func(cmd *cobra.Command, args []string) {

		componentSet := stringset.New(componentArr...)

		// TODO: Remove this in the future, while service-discovery is marked as
		//       deprecated we should still provide a consistent broker config file
		if !serviceDiscoveryEnabled {
			componentSet.Remove(components.ServiceDiscovery)
		}
		if globalnetEnable {
			componentSet.Add(components.Globalnet)
		}

		if err := isValidComponents(componentSet); err != nil {
			exitOnError("Invalid components parameter", err)
		}

		if valid, err := isValidGlobalnetConfig(); !valid {
			exitOnError("Invalid GlobalCIDR configuration", err)
		}
		config, err := getRestConfig(kubeConfig, kubeContext)
		exitOnError("The provided kubeconfig is invalid", err)

		status := cli.NewStatus()

		status.Start("Setting up broker RBAC")
		err = broker.Ensure(config, componentArr, false)
		status.End(cli.CheckForError(err))
		exitOnError("Error setting up broker RBAC", err)

		status.Start("Deploying the Submariner operator")
		err = submarinerop.Ensure(status, config, OperatorNamespace, operatorImage())
		status.End(cli.CheckForError(err))
		exitOnError("Error deploying the operator", err)

		status.Start("Deploying the broker")
		err = brokercr.Ensure(config, OperatorNamespace, populateBrokerSpec())
		if err == nil {
			status.QueueSuccessMessage("The broker has been deployed")
			status.End(cli.Success)
		} else {
			status.QueueFailureMessage("Broker deployment failed")
			status.End(cli.Failure)
		}
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

		subctlData.ServiceDiscovery = serviceDiscoveryEnabled
		subctlData.SetComponents(componentSet)

		if len(defaultCustomDomains) > 0 {
			subctlData.CustomDomains = &defaultCustomDomains
		}

		exitOnError("Error setting up service discovery information", err)

		err = broker.CreateGlobalnetConfigMap(config, globalnetEnable, globalnetCIDRRange,
			defaultGlobalnetClusterSize, broker.SubmarinerBrokerNamespace)
		exitOnError("Error creating globalCIDR configmap on Broker", err)

		err = subctlData.WriteToFile(brokerDetailsFilename)
		status.End(cli.CheckForError(err))
		exitOnError("Error writing the broker information", err)

	},
}

func isValidComponents(componentSet stringset.Interface) error {
	validComponentSet := stringset.New(validComponents...)

	if componentSet.Size() < 1 {
		return fmt.Errorf("at least one component must be provided for deployment")
	}

	for _, component := range componentSet.Elements() {
		if !validComponentSet.Contains(component) {
			return fmt.Errorf("unknown component: %s", component)
		}
	}

	return nil
}

func isValidGlobalnetConfig() (bool, error) {
	var err error
	if !globalnetEnable {
		return true, nil
	}
	defaultGlobalnetClusterSize, err = globalnet.GetValidClusterSize(globalnetCIDRRange, defaultGlobalnetClusterSize)
	if err != nil || defaultGlobalnetClusterSize == 0 {
		return false, err
	}
	return true, err
}

func populateBrokerSpec() submarinerv1a1.BrokerSpec {
	brokerSpec := submarinerv1a1.BrokerSpec{
		GlobalnetEnabled:            globalnetEnable,
		GlobalnetCIDRRange:          globalnetCIDRRange,
		DefaultGlobalnetClusterSize: defaultGlobalnetClusterSize,
		Components:                  componentArr,
		DefaultCustomDomains:        defaultCustomDomains,
	}
	return brokerSpec
}
