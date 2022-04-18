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
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
)

func Connections(clusterInfo *cluster.Info, status reporter.Interface) bool {
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

	tracker := reporter.NewTracker(status)
	foundActive := false

	for i := range gateways {
		gateway := &gateways[i]
		if gateway.Status.HAStatus != submv1.HAStatusActive {
			continue
		}

		foundActive = true

		if len(gateway.Status.Connections) == 0 {
			tracker.Failure("There are no active connections on gateway %q", gateway.Name)
		}

		for j := range gateway.Status.Connections {
			connection := &gateway.Status.Connections[j]
			if connection.Status == submv1.Connecting {
				tracker.Failure("Connection to cluster %q is in progress", connection.Endpoint.ClusterID)
			} else if connection.Status == submv1.ConnectionError {
				tracker.Failure("Connection to cluster %q is not established", connection.Endpoint.ClusterID)
			}
		}
	}

	if !foundActive {
		tracker.Failure("No active gateway was found")
	}

	if tracker.HasFailures() {
		return false
	}

	status.Success("All connections are established")

	return true
}
