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
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func discoverGenericNetwork(clientSet kubernetes.Interface) (*ClusterNetwork, error) {
	clusterNetwork := &ClusterNetwork{
		NetworkPlugin: "generic",
	}

	podIPRange, err := findPodIPRange(clientSet)
	if err != nil {
		return nil, err
	}

	if podIPRange != "" {
		clusterNetwork.PodCIDRs = []string{podIPRange}
	}

	clusterIPRange, err := findClusterIPRange(clientSet)
	if err != nil {
		return nil, err
	}

	if clusterIPRange != "" {
		clusterNetwork.ServiceCIDRs = []string{clusterIPRange}
	}

	if len(clusterNetwork.PodCIDRs) > 0 || len(clusterNetwork.ServiceCIDRs) > 0 {
		return clusterNetwork, nil
	}

	return nil, nil
}

func findClusterIPRange(clientSet kubernetes.Interface) (string, error) {
	clusterIPRange, err := findClusterIPRangeFromApiserver(clientSet)
	if err != nil || clusterIPRange != "" {
		return clusterIPRange, err
	}

	return "", nil
}

func findClusterIPRangeFromApiserver(clientSet kubernetes.Interface) (string, error) {
	return findPodCommandParameter(clientSet, "component=kube-apiserver", "--service-cluster-ip-range")
}

func findPodIPRange(clientSet kubernetes.Interface) (string, error) {
	podIPRange, err := findPodIPRangeKubeController(clientSet)
	if err != nil || podIPRange != "" {
		return podIPRange, err
	}

	podIPRange, err = findPodIPRangeKubeProxy(clientSet)
	if err != nil || podIPRange != "" {
		return podIPRange, err
	}

	podIPRange, err = findPodIPRangeFromNodeSpec(clientSet)
	if err != nil || podIPRange != "" {
		return podIPRange, err
	}

	return "", nil
}

func findPodIPRangeKubeController(clientSet kubernetes.Interface) (string, error) {
	return findPodCommandParameter(clientSet, "component=kube-controller-manager", "--cluster-cidr")
}

func findPodIPRangeKubeProxy(clientSet kubernetes.Interface) (string, error) {
	return findPodCommandParameter(clientSet, "component=kube-proxy", "--cluster-cidr")
}

func findPodIPRangeFromNodeSpec(clientSet kubernetes.Interface) (string, error) {
	nodes, err := clientSet.CoreV1().Nodes().List(v1meta.ListOptions{})

	if err != nil {
		return "", errors.WithMessagef(err, "error listing nodes")
	}

	return parseToPodCidr(nodes.Items)
}

func parseToPodCidr(nodes []v1.Node) (string, error) {
	for _, node := range nodes {
		if node.Spec.PodCIDR != "" {
			return node.Spec.PodCIDR, nil
		}
	}

	return "", nil
}
