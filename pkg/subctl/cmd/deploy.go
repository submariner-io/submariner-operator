package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
)

var (
	brokerKubeConfig    string
	brokerKubeContext   string
	clusterKubeConfigs  []string
	clusterKubeContexts []string
	clusterIDs          []string
)

func init() {
	deployCmd.Flags().StringVar(&brokerKubeConfig, "broker-kubeconfig", "",
		"absolute path(s) to the broker kubeconfig file(s)")
	deployCmd.Flags().StringVar(&brokerKubeContext, "broker-kubecontext", "",
		"kubeconfig context to use")
	deployCmd.Flags().StringSliceVar(&clusterKubeConfigs, "cluster-kubeconfigs", []string{},
		"absolute path(s) to the cluster kubeconfig file(s), comma-separated")
	deployCmd.Flags().StringSliceVar(&clusterKubeContexts, "cluster-kubecontexts", []string{},
		"kubeconfig contexts to use, comma-separated")
	deployCmd.Flags().StringSliceVar(&clusterIDs, "clusterids", []string{},
		"cluster IDs used to identify the tunnels, comma-separated")
	addJoinFlags(deployCmd)
	rootCmd.AddCommand(deployCmd)
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploy Submariner on a broker and connecting clusters",
	Run: func(cmd *cobra.Command, args []string) {
		brokerConfig, err := getRestConfig(brokerKubeConfig, brokerKubeContext)
		panicOnError(err)

		fmt.Printf("* Deploying broker\n")
		err = broker.Ensure(brokerConfig, IPSECPSKBytes)
		panicOnError(err)

		subctlData, err := datafile.NewFromCluster(brokerConfig, broker.SubmarinerBrokerNamespace)
		panicOnError(err)

		fmt.Printf("Writing submariner broker data to %s\n", brokerDetailsFilename)
		err = subctlData.WriteToFile(brokerDetailsFilename)
		panicOnError(err)

		if len(clusterKubeConfigs) > 0 {
			if len(clusterKubeConfigs) == len(clusterIDs) {
				for i := range clusterKubeConfigs {
					// Do we have a context?
					context := ""
					if i < len(clusterKubeContexts) {
						context = clusterKubeContexts[i]
					}
					fmt.Printf("Deploying Submariner using config %s and context %s\n", clusterKubeConfigs[i], context)
					clusterConfig, err := getRestConfig(clusterKubeConfigs[i], context)
					panicOnError(err)
					joinSubmarinerCluster(clusterConfig, clusterIDs[i], subctlData)
				}
			}
		}
	},
}
