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

	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
)

func Network(newCluster *cluster.Info, status reporter.Interface) bool {
	status.Start("Showing Network details")

	var clusterNetwork *network.ClusterNetwork
	var msg string

	if newCluster.Submariner != nil {
		msg = "    Discovered network details via Submariner:"
		clusterNetwork = &network.ClusterNetwork{
			PodCIDRs:      []string{newCluster.Submariner.Status.ClusterCIDR},
			ServiceCIDRs:  []string{newCluster.Submariner.Status.ServiceCIDR},
			NetworkPlugin: newCluster.Submariner.Status.NetworkPlugin,
			GlobalCIDR:    newCluster.Submariner.Status.GlobalCIDR,
		}
	} else {
		msg = "    Discovered network details"

		var err error
		clusterNetwork, err = network.Discover(newCluster.ClientProducer.ForDynamic(), newCluster.ClientProducer.ForKubernetes(),
			newCluster.ClientProducer.ForOperator(), constants.OperatorNamespace)
		if err != nil {
			status.Failure("There was an error discovering network details for this cluster %s", err)
			return false
		}
	}

	status.End()

	if clusterNetwork != nil {
		fmt.Println(msg)
	}

	clusterNetwork.Show()

	return true
}
