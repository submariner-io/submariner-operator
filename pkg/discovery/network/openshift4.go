package network

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog"
)

var (
	openshift4clusterNetworkGVR = schema.GroupVersionResource{
		Group:    "network.openshift.io",
		Version:  "v1",
		Resource: "clusternetworks",
	}
)

func discoverOpenShift4Network(dynClient dynamic.Interface) (*ClusterNetwork, error) {

	crClient := dynClient.Resource(openshift4clusterNetworkGVR)

	cr, err := crClient.Get("default", metav1.GetOptions{})
	if err != nil {
		klog.Info("Attempted network discovery for OpenShift4, no clusternetworks CRD")
		return nil, nil
	}

	return parseOS4ClusterNetwork(cr)
}

func parseOS4ClusterNetwork(cr *unstructured.Unstructured) (*ClusterNetwork, error) {

	result := &ClusterNetwork{}
	clusterNetworks, found, err := unstructured.NestedSlice(cr.Object, "clusterNetworks")
	if err != nil {
		return nil, err
	} else if !found {
		return nil, fmt.Errorf("field clusterNetworks expected, but not found in %v", cr.Object)
	}
	for _, clusterNetwork := range clusterNetworks {
		clusterNetworkMap, _ := clusterNetwork.(map[string]interface{})
		cidr, found, err := unstructured.NestedString(clusterNetworkMap, "CIDR")

		if err != nil {
			return nil, err
		} else if !found {
			return nil, fmt.Errorf("field CIDR expected, but not found in %v", clusterNetworkMap)
		}
		result.PodCIDRs = append(result.PodCIDRs, cidr)
	}
	serviceNetwork, found, err := unstructured.NestedString(cr.Object, "serviceNetwork")
	if err != nil {
		return nil, err
	} else if !found {
		return nil, fmt.Errorf("field serviceNetwork expected, but not found in %v", cr.Object)
	}
	result.ServiceCIDRs = append(result.ServiceCIDRs, serviceNetwork)
	result.NetworkPlugin = "OpenShift"
	return result, nil
}
