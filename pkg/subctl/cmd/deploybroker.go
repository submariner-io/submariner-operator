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
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
)

var disableDataplane bool

func init() {
	deployBroker.PersistentFlags().BoolVarP(&disableDataplane, "no-dataplane", "n", false,
		"Don't install the submariner dataplane on the broker")
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
		panicOnError(err)

		fmt.Printf("* Deploying broker\n")
		err = broker.Ensure(config, IPSECPSKBytes)
		panicOnError(err)

		subctlData, err := datafile.NewFromCluster(config, broker.SubmarinerBrokerNamespace)
		panicOnError(err)

		fmt.Printf("Writing submariner broker data to %s\n", brokerDetailsFilename)
		err = subctlData.WriteToFile(brokerDetailsFilename)
		panicOnError(err)

		if !disableDataplane {
			joinSubmarinerCluster(subctlData)
		}
	},
}
