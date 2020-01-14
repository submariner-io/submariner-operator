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
)

var enableDataplane bool
var disableDataplane bool
var enableLighthouse bool

func init() {
	deployBroker.PersistentFlags().BoolVar(&enableDataplane, "dataplane", false,
		"Install the Submariner dataplane on the broker")
	deployBroker.PersistentFlags().BoolVar(&disableDataplane, "no-dataplane", true,
		"Don't install the Submariner dataplane on the broker (default)")
	deployBroker.PersistentFlags().BoolVar(&enableLighthouse, "service-discovery", false,
		"Enable Multi Cluster Service Discovery")
	err := deployBroker.PersistentFlags().MarkHidden("no-dataplane")
	// An error here indicates a programming error (the argument isn’t declared), panic
	panicOnError(err)
	addJoinFlags(deployBroker)
	rootCmd.AddCommand(deployBroker)
}

const IPSECPSKBytes = 48 // using base64 this results on a 64 character password
const brokerDetailsFilename = "broker-info.subm"

var deployBroker = &cobra.Command{
	Use:   "deploy-broker",
	Short: "set the broker up",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := getRestConfig()
		exitOnError("The provided kubeconfig is invalid", err)

		status := cli.NewStatus()

		status.Start("Deploying broker")
		err = broker.Ensure(config, IPSECPSKBytes)
		status.End(err == nil)
		exitOnError("Error deploying the broker", err)

		subctlData, err := datafile.NewFromCluster(config, broker.SubmarinerBrokerNamespace, enableLighthouse)
		exitOnError("Error retrieving the broker information", err)

		fmt.Printf("Writing submariner broker data to %s\n", brokerDetailsFilename)
		err = subctlData.WriteToFile(brokerDetailsFilename)
		exitOnError("Error writing the broker information", err)

		if enableDataplane {
			joinSubmarinerCluster(subctlData)
		}
	},
}
