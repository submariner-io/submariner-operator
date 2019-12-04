/*
© 2019 Red Hat, Inc. and others.

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
	"k8s.io/client-go/kubernetes"
)

func discoverWeaveNetwork(clientSet kubernetes.Interface) (*ClusterNetwork, error) {

	weaveNetPod, err := findPod(clientSet, "name=weave-net")

	if err != nil || weaveNetPod == nil {
		return nil, err
	}

	var clusterNetwork *ClusterNetwork

	for _, container := range weaveNetPod.Spec.Containers {
		for _, envVar := range container.Env {
			if envVar.Name == "IPALLOC_RANGE" {
				clusterNetwork = &ClusterNetwork{
					PodCIDRs:      []string{envVar.Value},
					NetworkPlugin: "weave-net",
				}
				break
			}
		}
	}

	if clusterNetwork == nil {
		return nil, nil
	}

	clusterIPRange, err := findClusterIPRange(clientSet)
	if err == nil && clusterIPRange != "" {
		clusterNetwork.ServiceCIDRs = []string{clusterIPRange}
	}

	return clusterNetwork, nil
}
