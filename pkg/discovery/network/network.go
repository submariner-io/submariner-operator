package network

import "k8s.io/client-go/dynamic"

type ClusterNetwork struct {
	PodCIDRs     []string
	ServiceCIDRs []string
}

func Discover(dynClient dynamic.Interface) (*ClusterNetwork, error) {
	//TODO: Add more discovery mechanisms here
	return discoverOpenShift4Network(dynClient)
}
