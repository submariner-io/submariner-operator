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

package diagnose

import (
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
)

func Connections(clusterInfo *cluster.Info, status reporter.Interface) bool {
	if clusterInfo.Submariner == nil {
		status.Warning(constants.SubmMissingMessage)

		return true
	}

	status.Start("Checking gateway connections")
	defer status.End()

	gateways, err := clusterInfo.GetGateways()
	if err != nil {
		status.Failure("Error retrieving gateways: %v", err)

		return false
	}

	if len(gateways) == 0 {
		status.Failure("There are no gateways detected")

		return false
	}

	foundActive := false
	failed := false

	for i := range gateways {
		gateway := &gateways[i]
		if gateway.Status.HAStatus != submv1.HAStatusActive {
			continue
		}

		foundActive = true

		if len(gateway.Status.Connections) == 0 {
			status.Failure("There are no active connections on gateway %q", gateway.Name)

			failed = true
		}

		for j := range gateway.Status.Connections {
			connection := &gateway.Status.Connections[j]

			if connection.Status == submv1.Connecting {
				status.Failure("Connection to cluster %q is in progress", connection.Endpoint.ClusterID)

				failed = true
			} else if connection.Status == submv1.ConnectionError {
				status.Failure("Connection to cluster %q is not established", connection.Endpoint.ClusterID)

				failed = true
			}
		}
	}

	if !foundActive {
		status.Failure("No active gateway was found")
		status.End()
	}

	if failed {
		return false
	}

	status.Success("All connections are established")

	return true
}
