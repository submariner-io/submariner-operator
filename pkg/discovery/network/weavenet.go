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

package network

import (
	"fmt"

	"github.com/submariner-io/submariner/pkg/routeagent_driver/constants"
	"k8s.io/client-go/kubernetes"
)

// nolint:nilnil // Intentional as the purpose is to discover.
func discoverWeaveNetwork(clientSet kubernetes.Interface) (*ClusterNetwork, error) {
	weaveNetPod, err := FindPod(clientSet, "name=weave-net")

	if err != nil || weaveNetPod == nil {
		return nil, err
	}

	var clusterNetwork *ClusterNetwork

	for i := range weaveNetPod.Spec.Containers {
		for _, envVar := range weaveNetPod.Spec.Containers[i].Env {
			if envVar.Name == "IPALLOC_RANGE" {
				clusterNetwork = &ClusterNetwork{
					PodCIDRs:      []string{envVar.Value},
					NetworkPlugin: constants.NetworkPluginWeaveNet,
				}

				break
			}
		}
	}

	if clusterNetwork == nil {
		return nil, nil
	}

	clusterIPRange, clusterIPSource, err := findClusterIPRange(clientSet)
	if err == nil && clusterIPRange != "" {
		clusterNetwork.ServiceCIDRs = []string{clusterIPRange}
		clusterNetwork.InfoSrc += fmt.Sprintf(" clusterIPSource: %s", clusterIPSource)
	}

	return clusterNetwork, nil
}
