package network

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type ClusterNetwork struct {
	PodCIDRs      []string
	ServiceCIDRs  []string
	NetworkPlugin string
}

func Discover(dynClient dynamic.Interface, clientSet kubernetes.Interface) (*ClusterNetwork, error) {

	osClusterNet, err := discoverOpenShift4Network(dynClient)
	if err == nil && osClusterNet != nil {
		return osClusterNet, nil
	}

	weaveClusterNet, err := discoverWeaveNetwork(clientSet)
	if err == nil && weaveClusterNet != nil {
		return weaveClusterNet, nil
	}

	genericNet, err := discoverGenericNetwork(clientSet)
	if err == nil && genericNet != nil {
		return genericNet, nil
	}

	return nil, nil
}
