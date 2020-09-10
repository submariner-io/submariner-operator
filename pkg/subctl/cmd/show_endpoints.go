package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
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

func getEndpointsStatus(config *rest.Config) []endpointStatus {
	submarinerClient, err := submarinerclientset.NewForConfig(config)
	exitOnError("Unable to get the Submariner client", err)

	var status []endpointStatus

	existingCfg, err := submarinerClient.SubmarinerV1alpha1().Submariners(OperatorNamespace).Get(submarinercr.SubmarinerName, v1.GetOptions{})
	if err != nil {
		exitOnError("Error obtaining the Submariner resource", err)
	}

	gateways := existingCfg.Status.Gateways
	if gateways == nil {
		exitWithErrorMsg("No endpoints found")
	}

	for _, gateway := range *gateways {
		status = append(status, newEndpointsStatusFrom(
			gateway.Status.LocalEndpoint.ClusterID,
			gateway.Status.LocalEndpoint.PrivateIP,
			gateway.Status.LocalEndpoint.PublicIP,
			gateway.Status.LocalEndpoint.Backend,
			"local"))

		for _, connection := range gateway.Status.Connections {
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
		status := getEndpointsStatus(item.config)
		printEndpoints(status)
	}
}

func showEndpointsFromConfig(config *rest.Config) {
	status := getEndpointsStatus(config)
	printEndpoints(status)
}

func printEndpoints(endpoints []endpointStatus) {
	if len(endpoints) == 0 {
		fmt.Println("No resources found.")
		return
	}

	template := "%-20s%-16s%-16s%-20s%-16s\n"

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
