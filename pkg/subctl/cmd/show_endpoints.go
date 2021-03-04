/*
© 2021 Red Hat, Inc. and others

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
)

type endpointStatus struct {
	clusterID    string
	endpointIP   string
	publicIP     string
	cableDriver  string
	endpointType string
}

func newEndpointsStatusFrom(clusterID, endpointIP, publicIP, cableDriver, endpointType string) endpointStatus {
	return endpointStatus{
		clusterID:    clusterID,
		endpointIP:   endpointIP,
		publicIP:     publicIP,
		cableDriver:  cableDriver,
		endpointType: endpointType,
	}
}

var showEndpointsCmd = &cobra.Command{
	Use:     "endpoints",
	Short:   "Show submariner endpoint information",
	Long:    `This command shows information about submariner endpoints in a cluster.`,
	PreRunE: checkVersionMismatch,
	Run:     showEndpoints,
}

func init() {
	showCmd.AddCommand(showEndpointsCmd)
}

func getEndpointsStatus(submariner *v1alpha1.Submariner) []endpointStatus {
	gateways := submariner.Status.Gateways

	if gateways == nil {
		exitWithErrorMsg("No endpoints found")
	}

	var status = make([]endpointStatus, 0, len(*gateways))

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
			item.clusterID,
			item.endpointIP,
			item.publicIP,
			item.cableDriver,
			item.endpointType)
	}
}
