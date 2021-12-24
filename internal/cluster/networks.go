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

package cluster

import (
	"fmt"

	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
)

func GetNetworkDetails(clientProducer client.Producer, status reporter.Interface) (*network.ClusterNetwork, error) {
	networkDetails, err := network.Discover(clientProducer.ForDynamic(), clientProducer.ForKubernetes(), clientProducer.ForOperator(),
		constants.OperatorNamespace)
	if err != nil {
		status.Warning(fmt.Sprintf("Error trying to discover network details: %s", err))
	} else if networkDetails != nil {
		networkDetails.Show()
	}

	return networkDetails, nil
}

func GetPodCIDR(clusterCIDR string, nd *network.ClusterNetwork, status reporter.Interface) (cidrType string, autodetected bool, err error) {
	if clusterCIDR != "" {
		if nd != nil && len(nd.PodCIDRs) > 0 && nd.PodCIDRs[0] != clusterCIDR {
			status.Warning(fmt.Sprintf("Your provided cluster CIDR for the pods (%s) does not match discovered (%s)\n",
				clusterCIDR, nd.PodCIDRs[0]))
		}

		return clusterCIDR, false, nil
	} else if nd != nil && len(nd.PodCIDRs) > 0 {
		return nd.PodCIDRs[0], true, nil
	} else {
		return "", false, fmt.Errorf("could not determine cluster network")
	}
}

func GetServiceCIDR(serviceCIDR string, nd *network.ClusterNetwork, status reporter.Interface) (cidrType string,
	autodetected bool, err error) {
	if serviceCIDR != "" {
		if nd != nil && len(nd.ServiceCIDRs) > 0 && nd.ServiceCIDRs[0] != serviceCIDR {
			status.Warning(fmt.Sprintf("Your provided service CIDR (%s) does not match discovered (%s)\n",
				serviceCIDR, nd.ServiceCIDRs[0]))
		}

		return serviceCIDR, false, nil
	} else if nd != nil && len(nd.ServiceCIDRs) > 0 {
		return nd.ServiceCIDRs[0], true, nil
	}

	return "", false, fmt.Errorf("could not determine cluster network")
}
