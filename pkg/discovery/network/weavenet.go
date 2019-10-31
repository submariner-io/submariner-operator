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
