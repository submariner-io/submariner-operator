package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install"
)

var disableDataplane bool

func init() {
	deployBroker.PersistentFlags().BoolVarP(&disableDataplane, "no-dataplane", "n", false,
		"Don't install the submariner dataplane on the broker")
	rootCmd.AddCommand(deployBroker)
}

// FIXME: Use this var everywhere
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
			err = handleNodeLabels()
			panicOnError(err)

			//TODO: when operator handles broker, we will also need to install the operator
			//      but make sure the operator is able to not-handle dataplane, just broker
			fmt.Printf("* Deploying the submariner operator\n")
			err = install.Ensure(config, OperatorNamespace, operatorImage)
			panicOnError(err)
		}
	},
}
