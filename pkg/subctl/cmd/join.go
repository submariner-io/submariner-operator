package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install"
)

func init() {
	rootCmd.AddCommand(joinCmd)
}

var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "connect a cluster to an existing broker",
	Args:  cobra.ExactArgs(1), // exactly one, the broker data file
	Run: func(cmd *cobra.Command, args []string) {

		config, err := getRestConfig()
		panicOnError(err)

		err = handleNodeLabels()
		panicOnError(err)

		subctlData, err := datafile.NewFromFile(args[0])
		panicOnError(err)
		fmt.Printf("* %s says broker is at: %s\n", args[0], subctlData.BrokerURL)

		fmt.Printf("* Deploying the submariner operator\n")
		err = install.Ensure(config, OperatorNamespace, operatorImage)
		panicOnError(err)
	},
}
