package cmd

import (
	"fmt"
	"regexp"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
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
	addJoinFlags(joinCmd)
	rootCmd.AddCommand(joinCmd)

}

func addJoinFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&clusterID, "clusterid", "", "cluster ID used to identify the tunnels")
	cmd.Flags().StringVar(&serviceCIDR, "servicecidr", "", "service CIDR")
	cmd.Flags().StringVar(&clusterCIDR, "clustercidr", "", "cluster CIDR")
	cmd.Flags().StringVar(&repository, "repository", "", "image repository")
	cmd.Flags().StringVar(&version, "version", "", "image version")
	cmd.Flags().StringVarP(&operatorImage, "operator-image", "o", DefaultOperatorImage,
		"the operator image you wish to use")
}

const (
	SubmarinerNamespace = "submariner-operator" // We currently expect everything in submariner-operator
)

var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "connect a cluster to an existing broker",
	Args:  cobra.ExactArgs(1), // exactly one, the broker data file
	Run: func(cmd *cobra.Command, args []string) {
		// Make sure the clusterid is a valid DNS-1123 string
		if match, _ := regexp.MatchString("^[a-z0-9][a-z0-9.-]*[a-z0-9]$", clusterID); !match {
			fmt.Printf("Cluster IDs must be valid DNS-1123 names, with only lowercase alphanumerics,\n"+
				"'.' or '-' (and the first and last characters must be alphanumerics).\n"+
				"%s doesn't meet these requirements\n", clusterID)
			return
		}

		subctlData, err := datafile.NewFromFile(args[0])
		panicOnError(err)
		fmt.Printf("* %s says broker is at: %s\n", args[0], subctlData.BrokerURL)
		joinSubmarinerCluster(subctlData)
	},
}

func joinSubmarinerCluster(subctlData *datafile.SubctlData) {

	config, err := getRestConfig()
	panicOnError(err)

	err = handleNodeLabels()
	panicOnError(err)

	fmt.Printf("* Deploying the submariner operator\n")
	err = install.Ensure(config, OperatorNamespace, operatorImage)
	panicOnError(err)

	fmt.Printf("* Discovering network details\n")
	networkDetails := getNetworkDetails()

	serviceCIDR, err = getServiceCIDR(serviceCIDR, networkDetails)
	panicOnError(err)

	clusterCIDR, err = getPodCIDR(clusterCIDR, networkDetails)
	panicOnError(err)

	fmt.Printf("* Deploying Submariner\n")
	err = deploy.Ensure(config, SubmarinerNamespace, repository, version, clusterID, serviceCIDR, clusterCIDR, subctlData)
	panicOnError(err)
}

func getNetworkDetails() *network.ClusterNetwork {

	dynClient, clientSet, err := getClients()
	panicOnError(err)
	networkDetails, err := network.Discover(dynClient, clientSet)
	if err != nil {
		fmt.Printf("Error trying to discover network details: %s\n", err)
	} else {
		networkDetails.Show()
	}
	return networkDetails
}

func getPodCIDR(clusterCIDR string, nd *network.ClusterNetwork) (string, error) {
	if clusterCIDR != "" {
		if nd != nil && len(nd.PodCIDRs) > 0 && nd.PodCIDRs[0] != clusterCIDR {
			fmt.Printf("WARNING: your provided cluster CIDR for the pods (%s) does not match discovered (%s)\n",
				clusterCIDR, nd.PodCIDRs[0])
		}
		return clusterCIDR, nil
	} else if len(nd.PodCIDRs) > 0 {
		return nd.PodCIDRs[0], nil
	} else {
		return askForCIDR("Pod")
	}
}

func getServiceCIDR(serviceCIDR string, nd *network.ClusterNetwork) (string, error) {
	if serviceCIDR != "" {
		if nd != nil && len(nd.ServiceCIDRs) > 0 && nd.ServiceCIDRs[0] != serviceCIDR {
			fmt.Printf("WARNING: your provided service CIDR (%s) does not match discovered (%s)\n",
				serviceCIDR, nd.ServiceCIDRs[0])
		}
		return serviceCIDR, nil
	} else if len(nd.ServiceCIDRs) > 0 {
		return nd.ServiceCIDRs[0], nil
	} else {
		return askForCIDR("ClusterIP service")
	}
}

func askForCIDR(name string) (string, error) {
	var qs = []*survey.Question{{
		Name:     "cidr",
		Prompt:   &survey.Input{Message: fmt.Sprintf("What's the %s CIDR for your cluster?", name)},
		Validate: survey.Required,
	}}

	answers := struct {
		Cidr string
	}{}

	err := survey.Ask(qs, &answers)
	if err != nil {
		return "", err
	} else {
		return answers.Cidr, nil
	}
}
