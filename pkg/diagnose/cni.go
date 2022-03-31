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

package diagnose

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	"github.com/submariner-io/submariner/pkg/routeagent_driver/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

var supportedNetworkPlugins = []string{
	constants.NetworkPluginGeneric, constants.NetworkPluginCanalFlannel, constants.NetworkPluginWeaveNet,
	constants.NetworkPluginOpenShiftSDN, constants.NetworkPluginOVNKubernetes, constants.NetworkPluginCalico,
}

var calicoGVR = schema.GroupVersionResource{
	Group:    "crd.projectcalico.org",
	Version:  "v1",
	Resource: "ippools",
}

func CNIConfig(clusterInfo *cluster.Info, status reporter.Interface) bool {
	mustHaveSubmariner(clusterInfo)

	status.Start("Checking Submariner support for the CNI network plugin")
	defer status.End()

	isSupportedPlugin := false

	for _, np := range supportedNetworkPlugins {
		if clusterInfo.Submariner.Status.NetworkPlugin == np {
			isSupportedPlugin = true
			break
		}
	}

	if !isSupportedPlugin {
		status.Failure("The detected CNI network plugin (%q) is not supported by Submariner."+
			" Supported network plugins: %v\n", clusterInfo.Submariner.Status.NetworkPlugin, supportedNetworkPlugins)
		return false
	}

	if clusterInfo.Submariner.Status.NetworkPlugin == constants.NetworkPluginGeneric {
		status.Warning("Submariner could not detect the CNI network plugin and is using (%q) plugin."+
			" It may or may not work.", clusterInfo.Submariner.Status.NetworkPlugin)
	} else {
		status.Success("The detected CNI network plugin (%q) is supported", clusterInfo.Submariner.Status.NetworkPlugin)
	}

	return checkCalicoIPPoolsIfCalicoCNI(clusterInfo, status)
}

func detectCalicoConfigMap(clientSet kubernetes.Interface) (bool, error) {
	cmList, err := clientSet.CoreV1().ConfigMaps(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "error listing ConfigMaps")
	}

	for i := range cmList.Items {
		if cmList.Items[i].Name == "calico-config" {
			return true, nil
		}
	}

	return false, nil
}

func checkCalicoIPPoolsIfCalicoCNI(info *cluster.Info, status reporter.Interface) bool {
	status.Start("Trying to detect the Calico ConfigMap")
	defer status.End()

	found, err := detectCalicoConfigMap(info.ClientProducer.ForKubernetes())
	if err != nil {
		status.Failure("Error trying to detect the Calico ConfigMap: %s", err)
		return false
	}

	if !found {
		return true
	}

	status.Start("Calico CNI detected, checking if the Submariner IPPool pre-requisites are configured")

	gateways, err := info.GetGateways()
	if err != nil {
		status.Failure("Error retrieving Gateways: %v", err)
		return false
	}

	if len(gateways) == 0 {
		status.Warning("There are no gateways detected on the cluster")
		return false
	}

	client := info.ClientProducer.ForDynamic().Resource(calicoGVR)

	ippoolList, err := client.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		status.Failure("Error obtaining IPPools: %v", err)
		return false
	}

	if len(ippoolList.Items) < 1 {
		status.Failure("Could not find any IPPools in the cluster")
		return false
	}

	tracker := reporter.NewTracker(status)
	ippools := make(map[string]unstructured.Unstructured)

	for _, pool := range ippoolList.Items {
		cidr, found, err := unstructured.NestedString(pool.Object, "spec", "cidr")
		if err != nil {
			tracker.Failure("Error extracting field cidr from IPPool %q", pool.GetName())
			continue
		}

		if !found {
			tracker.Failure("No CIDR found in IPPool %q", pool.GetName())
			continue
		}

		ippools[cidr] = pool
	}

	checkGatewaySubnets(gateways, ippools, tracker)

	return !tracker.HasFailures()
}

func checkGatewaySubnets(gateways []submv1.Gateway, ippools map[string]unstructured.Unstructured, status reporter.Interface) {
	for i := range gateways {
		gateway := &gateways[i]
		if gateway.Status.HAStatus != submv1.HAStatusActive {
			continue
		}

		for j := range gateway.Status.Connections {
			connection := &gateway.Status.Connections[j]
			for _, subnet := range connection.Endpoint.Subnets {
				ipPool, found := ippools[subnet]
				if found {
					isDisabled, err := getSpecBool(ipPool, "disabled")
					if err != nil {
						status.Failure(err.Error())
						continue
					}

					// When disabled is set to true, Calico IPAM will not assign addresses from this Pool.
					// The IPPools configured for Submariner remote CIDRs should have disabled as true.
					if !isDisabled {
						status.Failure("The IPPool %q with CIDR %q for remote endpoint"+
							" %q has disabled set to false", ipPool.GetName(), subnet, connection.Endpoint.CableName)
						continue
					}
				} else {
					status.Failure("Could not find any IPPool with CIDR %q for remote"+
						" endpoint %q", subnet, connection.Endpoint.CableName)
					continue
				}
			}
		}
	}
}

func getSpecBool(pool unstructured.Unstructured, key string) (bool, error) {
	isDisabled, found, err := unstructured.NestedBool(pool.Object, "spec", key)
	if err != nil {
		return false, errors.Wrap(err, "error getting spec field")
	}

	if !found {
		return false, fmt.Errorf("%s status not found for IPPool %q", key, pool.GetName())
	}

	return isDisabled, nil
}

func mustHaveSubmariner(clusterInfo *cluster.Info) {
	if clusterInfo.Submariner == nil {
		panic("cluster.Info.Submariner field cannot be nil")
	}
}
