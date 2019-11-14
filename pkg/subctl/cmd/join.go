package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/deploy"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install"
)

var (
	clusterID   string
	serviceCIDR string
	clusterCIDR string
	repository  string
	version     string
)

func init() {
	rootCmd.AddCommand(joinCmd)
	joinCmd.Flags().StringVar(&clusterID, "clusterid", "", "cluster ID used to identify the tunnels")
	joinCmd.Flags().StringVar(&serviceCIDR, "servicecidr", "", "service CIDR")
	joinCmd.Flags().StringVar(&clusterCIDR, "clustercidr", "", "cluster CIDR")
	joinCmd.Flags().StringVar(&repository, "repository", "", "image repository")
	joinCmd.Flags().StringVar(&version, "version", "", "image version")
}

const (
	SubmarinerNamespace = "submariner-operator" // We currently expect everything in submariner-operator
)

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

		fmt.Printf("* Deploying Submariner\n")
		err = deploy.Ensure(config, SubmarinerNamespace, repository, version, clusterID, serviceCIDR, clusterCIDR, subctlData)
		panicOnError(err)
	},
}
