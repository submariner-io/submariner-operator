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

func discoverGenericNetwork(clientSet kubernetes.Interface) (*ClusterNetwork, error) {
	clusterNetwork := &ClusterNetwork{
		NetworkPlugin: "generic",
	}

	podIPRange, err := findPodIPRangeKubeController(clientSet)
	if err != nil {
		return nil, err
	}

	if podIPRange == "" {
		podIPRange, err = findPodIPRangeKubeProxy(clientSet)
		if err != nil {
			return nil, err
		}
	}

	if podIPRange != "" {
		clusterNetwork.PodCIDRs = []string{podIPRange}
	}

	// on some self-hosted platforms, the platform itself will provide the kube-apiserver, thus
	// our discovery method of looking for the kube-apiserver pod is useless, and we won't be
	// able to return such detail
	clusterIPRange, err := findClusterIPRange(clientSet)

	if err != nil {
		return nil, err
	}

	if clusterIPRange != "" {
		clusterNetwork.ServiceCIDRs = []string{clusterIPRange}
	}

	if len(clusterNetwork.PodCIDRs) > 0 || len(clusterNetwork.ServiceCIDRs) > 0 {
		return clusterNetwork, err
	}

	return nil, nil
}

func findClusterIPRange(clientSet kubernetes.Interface) (string, error) {
	return findPodCommandParameter(clientSet, "component=kube-apiserver", "--service-cluster-ip-range")
}

func findPodIPRangeKubeController(clientSet kubernetes.Interface) (string, error) {
	return findPodCommandParameter(clientSet, "component=kube-controller-manager", "--cluster-cidr")
}

func findPodIPRangeKubeProxy(clientSet kubernetes.Interface) (string, error) {
	return findPodCommandParameter(clientSet, "component=kube-proxy", "--cluster-cidr")
}
