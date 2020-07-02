package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type gatewaysStatus struct {
	node        string
	haStatus    submv1.HAStatus
	connections string
	summary     string
}

func newGatewayStatusFrom(node string, haStatus submv1.HAStatus, connections string, summary string) gatewaysStatus {
	v := gatewaysStatus{
		node:        node,
		haStatus:    haStatus,
		connections: connections,
		summary:     summary,
	}
	return v
}

var showGatewaysCmd = &cobra.Command{
	Use:   "gateways",
	Short: "Get information on your gateways related to submariner",
	Long:  `This command shows the status of submariner gateways in your cluster.`,
	Run:   showGateways,
}

func init() {
	showCmd.AddCommand(showGatewaysCmd)
}

func getGatewaysStatus(status []gatewaysStatus) []gatewaysStatus {
	config, err := getRestConfig(kubeConfig, kubeContext)
	exitOnError("Error getting REST config for cluster", err)

	submarinerClient, err := submarinerclientset.NewForConfig(config)
	exitOnError("Unable to get the Submariner client", err)

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
		if totalConnections > 0 && totalConnections == countConnected {
			summary = "All connections OK"
		} else {
			summary = gateway.Status.StatusFailure
		}
		connectionString := fmt.Sprintf("%d/%d", countConnected, totalConnections)
		status = append(status, newGatewayStatusFrom(enpoint, haStatus, connectionString, summary))
	}

	return status
}

func showGateways(cmd *cobra.Command, args []string) {
	var status []gatewaysStatus
	status = getGatewaysStatus(status)
	printGateways(status)
}

func printGateways(gateways []gatewaysStatus) {
	template := "%-20s%-16s%-16s%-32s\n"
	fmt.Printf(template, "NODE", "HA-STATUS", "CONNSCTIONS", "SUMMARY")
	for _, item := range gateways {
		fmt.Printf(
			template,
			item.node,
			item.haStatus,
			item.connections,
			item.summary)
	}
}
