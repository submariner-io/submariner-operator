package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
)

type endpointStatus struct {
	clusterId    string
	endpointIp   string
	publicIp     string
	cableDriver  string
	endpointType string
}

func newEndpointsStatusFrom(clusterId, endpointIp, publicIp, cableDriver, endpointType string) endpointStatus {
	return endpointStatus{
		clusterId:    clusterId,
		endpointIp:   endpointIp,
		publicIp:     publicIp,
		cableDriver:  cableDriver,
		endpointType: endpointType,
	}
}

var showEndpointsCmd = &cobra.Command{
	Use:   "endpoints",
	Short: "Show submariner endpoint information",
	Long:  `This command shows information about submariner endpoints in a cluster.`,
	Run:   showEndpoints,
}

func init() {
	showCmd.AddCommand(showEndpointsCmd)
}

func getEndpointsStatus(submariner *v1alpha1.Submariner) []endpointStatus {
	var status []endpointStatus
	gateways := submariner.Status.Gateways

	if gateways == nil {
		exitWithErrorMsg("No endpoints found")
	}

	for _, gateway := range *gateways {
		status = append(status, newEndpointsStatusFrom(
			gateway.LocalEndpoint.ClusterID,
			gateway.LocalEndpoint.PrivateIP,
			gateway.LocalEndpoint.PublicIP,
			gateway.LocalEndpoint.Backend,
			"local"))

		for _, connection := range gateway.Connections {
			status = append(status, newEndpointsStatusFrom(
				connection.Endpoint.ClusterID,
				connection.Endpoint.PrivateIP,
				connection.Endpoint.PublicIP,
				connection.Endpoint.Backend,
				"remote"))
		}
	}
	return status
}

func showEndpoints(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContext)
	exitOnError("Error getting REST config for cluster", err)
	for _, item := range configs {
		fmt.Println()
		fmt.Printf("Showing information for cluster %q:\n", item.clusterName)
		submariner := getSubmarinerResource(item.config)

		if submariner == nil {
			fmt.Println(submMissingMessage)
		} else {
			showEndpointsFor(submariner)
		}
	}
}

func showEndpointsFor(submariner *v1alpha1.Submariner) {
	status := getEndpointsStatus(submariner)
	printEndpoints(status)
}

func printEndpoints(endpoints []endpointStatus) {
	if len(endpoints) == 0 {
		fmt.Println("No resources found.")
		return
	}

	template := "%-30.29s%-16.15s%-16.15s%-20.19s%-16.15s\n"

	fmt.Printf(template, "CLUSTER ID", "ENDPOINT IP", "PUBLIC IP", "CABLE DRIVER", "TYPE")

	for _, item := range endpoints {
		fmt.Printf(
			template,
			item.clusterId,
			item.endpointIp,
			item.publicIp,
			item.cableDriver,
			item.endpointType)
	}
}
