/*
SPDX-License-Identifier: Apache-2.0

Copyright Contributors to the Submariner project.

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

package show

import (
	"fmt"

	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
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

func Endpoints(clusterInfo *cluster.Info, status reporter.Interface) bool {
	status.Start("Showing Endpoints")

	gateways, err := clusterInfo.GetGateways()
	if err != nil {
		status.Failure("Error retrieving gateways: %v", err)
		status.End()

		return false
	}

	if len(gateways) == 0 {
		status.Failure("There are no gateways detected")
		status.End()

		return false
	}

	epStatus := make([]endpointStatus, 0, len(gateways))

	for i := range gateways {
		gateway := &gateways[i]
		epStatus = append(epStatus, newEndpointsStatusFrom(
			gateway.Status.LocalEndpoint.ClusterID,
			gateway.Status.LocalEndpoint.PrivateIP,
			gateway.Status.LocalEndpoint.PublicIP,
			gateway.Status.LocalEndpoint.Backend,
			"local"))

		for i := range gateway.Status.Connections {
			connection := &gateway.Status.Connections[i]
			epStatus = append(epStatus, newEndpointsStatusFrom(
				connection.Endpoint.ClusterID,
				connection.Endpoint.PrivateIP,
				connection.Endpoint.PublicIP,
				connection.Endpoint.Backend,
				"remote"))
		}
	}

	if len(epStatus) == 0 {
		status.Failure("No Endpoints found")
		status.End()

		return false
	}

	status.End()
	printEndpoints(epStatus)

	return true
}

func printEndpoints(endpoints []endpointStatus) {
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
