package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type connectionStatus struct {
	gateway     string
	cluster     string
	remoteIp    string
	cableDriver string
	subnets     string
	status      submv1.ConnectionStatus
}

var showConnectionsCmd = &cobra.Command{
	Use:   "connections",
	Short: "Show cluster connectivity information",
	Long:  `This command shows information about submariner endpoint connections with other clusters.`,
	Run:   showConnections,
}

func init() {
	showCmd.AddCommand(showConnectionsCmd)
}

func getConnectionsStatus() []connectionStatus {
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
		exitWithErrorMsg("No endpoints found")
	}
	var status []connectionStatus
	for _, gateway := range *gateways {
		for _, connection := range gateway.Status.Connections {
			subnets := strings.Join(connection.Endpoint.Subnets, ", ")

			status = append(status, connectionStatus{
				gateway:     connection.Endpoint.Hostname,
				cluster:     connection.Endpoint.ClusterID,
				remoteIp:    connection.Endpoint.PrivateIP,
				cableDriver: connection.Endpoint.Backend,
				subnets:     subnets,
				status:      connection.Status,
			})
		}
	}

	return status
}

func showConnections(cmd *cobra.Command, args []string) {
	status := getConnectionsStatus()
	printConnections(status)
}

func printConnections(connections []connectionStatus) {
	template := "%-20s%-16s%-16s%-24s%-40s%-16s\n"

	fmt.Printf(template, "GATEWAY", "CLUSTER", "REMOTE IP", "CABLE DRIVER", "SUBNETS", "STATUS")

	for _, item := range connections {
		fmt.Printf(
			template,
			item.gateway,
			item.cluster,
			item.remoteIp,
			item.cableDriver,
			item.subnets,
			item.status)
	}
}
