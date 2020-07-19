package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

type gatewayStatus struct {
	node     string
	haStatus submv1.HAStatus
	summary  string
}

var showGatewaysCmd = &cobra.Command{
	Use:   "gateways",
	Short: "Show submariner gateway summary information",
	Long:  `This command shows summary information about the submariner gateways in a cluster.`,
	Run:   showGateways,
}

func init() {
	showCmd.AddCommand(showGatewaysCmd)
}

func getGatewaysStatus(config *rest.Config) []gatewayStatus {
	submarinerClient, err := submarinerclientset.NewForConfig(config)
	exitOnError("Unable to get the Submariner client", err)

	var status []gatewayStatus
	existingCfg, err := submarinerClient.SubmarinerV1alpha1().Submariners(OperatorNamespace).Get(submarinercr.SubmarinerName, v1.GetOptions{})
	if err != nil {
		exitOnError("error reading from submariner client", err)
	}

	gateways := existingCfg.Status.Gateways
	if gateways == nil {
		exitWithErrorMsg("no gateways found")
	}

	for _, gateway := range *gateways {
		haStatus := gateway.Status.HAStatus
		enpoint := gateway.Status.LocalEndpoint.Hostname
		totalConnections := len(gateway.Status.Connections)
		countConnected := 0
		for _, connection := range gateway.Status.Connections {
			if connection.Status == submv1.Connected {
				countConnected += 1
			}
		}

		var summary string
		if gateway.Status.StatusFailure != "" {
			summary = gateway.Status.StatusFailure
		} else if totalConnections == 0 {
			summary = "There are no connections"
		} else if totalConnections == countConnected {
			summary = fmt.Sprintf("All connections (%d) are established", totalConnections)
		} else {
			summary = fmt.Sprintf("%d connections out of %d are established", countConnected, totalConnections)
		}
		status = append(status,
			gatewayStatus{
				node:     enpoint,
				haStatus: haStatus,
				summary:  summary,
			})
	}

	return status
}

func showGateways(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContext)
	exitOnError("Error getting REST config for cluster", err)

	for _, item := range configs {
		fmt.Println()
		fmt.Printf("Showing information for cluster %q:\n", item.context)
		status := getGatewaysStatus(item.config)
		printGateways(status)
	}
}

func showGatewaysFromConfig(config *rest.Config) {
	status := getGatewaysStatus(config)
	printGateways(status)
}

func printGateways(gateways []gatewayStatus) {
	template := "%-20s%-16s%-32s\n"
	fmt.Printf(template, "NODE", "HA STATUS", "SUMMARY")
	for _, item := range gateways {
		fmt.Printf(
			template,
			item.node,
			item.haStatus,
			item.summary)
	}
}
