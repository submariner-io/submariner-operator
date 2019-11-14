package network

import (
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type ClusterNetwork struct {
	PodCIDRs      []string
	ServiceCIDRs  []string
	NetworkPlugin string
}

func (cn *ClusterNetwork) Show() {
	fmt.Printf("Discovered network details:\n")
	fmt.Printf("  Network plugin:  %s\n", cn.NetworkPlugin)
	fmt.Printf("  ClusterIP CIDRs: %v\n", cn.ServiceCIDRs)
	fmt.Printf("  Pod CIDRs:       %v\n", cn.PodCIDRs)
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
