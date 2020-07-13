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
	"fmt"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var (
	openshift4clusterNetworkGVR = schema.GroupVersionResource{
		Group:    "network.openshift.io",
		Version:  "v1",
		Resource: "clusternetworks",
	}
)

func discoverOpenShift4Network(dynClient dynamic.Interface) (*ClusterNetwork, error) {
	if dynClient == nil {
		return nil, nil
	}

	crClient := dynClient.Resource(openshift4clusterNetworkGVR)

	cr, err := crClient.Get("default", metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.WithMessage(err, "error obtaining the default OpenShift4 ClusterNetworks resource")
	}

	return parseOS4ClusterNetwork(cr)
}

func parseOS4ClusterNetwork(cr *unstructured.Unstructured) (*ClusterNetwork, error) {
	result := &ClusterNetwork{}
	clusterNetworks, found, err := unstructured.NestedSlice(cr.Object, "clusterNetworks")
	if err != nil {
		return nil, err
	} else if !found {
		return nil, fmt.Errorf("field clusterNetworks expected, but not found in ClusterNetworks resource: %v", cr.Object)
	}
	for _, clusterNetwork := range clusterNetworks {
		clusterNetworkMap, _ := clusterNetwork.(map[string]interface{})
		cidr, found, err := unstructured.NestedString(clusterNetworkMap, "CIDR")

		if err != nil {
			return nil, err
		} else if !found {
			return nil, fmt.Errorf("field CIDR expected, but not found in cluster network: %v", clusterNetworkMap)
		}
		result.PodCIDRs = append(result.PodCIDRs, cidr)
	}
	serviceNetwork, found, err := unstructured.NestedString(cr.Object, "serviceNetwork")
	if err != nil {
		return nil, err
	} else if !found {
		return nil, fmt.Errorf("field serviceNetwork expected, but not found in ClusterNetworks resource: %v", cr.Object)
	}
	result.ServiceCIDRs = append(result.ServiceCIDRs, serviceNetwork)
	result.NetworkPlugin = "OpenShift"
	return result, nil
}
