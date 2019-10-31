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
