package cmd

import (
	"github.com/spf13/cobra"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install"
)

// NOTE: Temporary command, to be moved to the join cluster / create broker commands

var installOperatorCmd = &cobra.Command{
	Use:   "install-operator",
	Short: "",
	Long:  `This command installs the operator, and should be removed down the line when we have broker/join commands`,
	Run:   installOperator,
}

func init() {
	rootCmd.AddCommand(installOperatorCmd)
}

func installOperator(cmd *cobra.Command, args []string) {

	config, err := getRestConfig()
	if err != nil {
		panic(err)
	}

	if err := install.Ensure(config, OperatorNamespace, operatorImage); err != nil {
		panic(err)
	}
}
