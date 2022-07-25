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
	"github.com/submariner-io/submariner/pkg/cni"
	"k8s.io/client-go/kubernetes"
)

func discoverWeaveNetwork(clientSet kubernetes.Interface) (*ClusterNetwork, error) {
	weaveNetPod, err := FindPod(clientSet, "name=weave-net")

	if err != nil || weaveNetPod == nil {
		return nil, err
	}

	clusterNetwork := &ClusterNetwork{
		NetworkPlugin: cni.WeaveNet,
	}

	for i := range weaveNetPod.Spec.Containers {
		for _, envVar := range weaveNetPod.Spec.Containers[i].Env {
			if envVar.Name == "IPALLOC_RANGE" {
				clusterNetwork.PodCIDRs = []string{envVar.Value}
				break
			}
		}
	}

	clusterIPRange, err := findClusterIPRange(clientSet)
	if err == nil && clusterIPRange != "" {
		clusterNetwork.ServiceCIDRs = []string{clusterIPRange}
	}

	return clusterNetwork, nil
}
