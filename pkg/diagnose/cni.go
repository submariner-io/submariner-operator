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
	operatorConstants "github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
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
	if clusterInfo.Submariner == nil {
		status.Warning(operatorConstants.SubmMissingMessage)

		return true
	}

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

	status.Success("The detected CNI network plugin (%q) is supported", clusterInfo.Submariner.Status.NetworkPlugin)

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

	status.Start("Calico CNI detected, checking if the Submariner IPPool pre-requisites are configured.")

	gateways, err := info.GetGateways()
	if err != nil {
		status.Failure("Error retrieving Gateways: %v", err)

		return false
	}

	if len(gateways) == 0 {
		status.Warning("There are no gateways detected on the resource")

		return false
	}

	client := info.ClientProducer.ForDynamic().Resource(calicoGVR)

	ippoolList, err := client.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		status.Failure("Error obtaining IPPools: %v", err)

		return false
	}

	if len(ippoolList.Items) < 1 {
		status.Failure("Could not find any IPPools in the resource")

		return false
	}

	ippools := make(map[string]unstructured.Unstructured)

	failed := false

	for _, pool := range ippoolList.Items {
		cidr, found, err := unstructured.NestedString(pool.Object, "spec", "cidr")
		if err != nil {
			status.Failure("Error extracting field cidr from IPPool %q", pool.GetName())

			failed = true

			continue
		}

		if !found {
			status.Failure("No CIDR found in IPPool %q", pool.GetName())

			failed = true

			continue
		}

		ippools[cidr] = pool
	}

	failed = failed || checkGatewaySubnets(gateways, ippools, status)

	return !failed
}

func checkGatewaySubnets(gateways []submv1.Gateway, ippools map[string]unstructured.Unstructured, status reporter.Interface) bool {
	failed := false

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

						failed = true

						continue
					}

					// When disabled is set to true, Calico IPAM will not assign addresses from this Pool.
					// The IPPools configured for Submariner remote CIDRs should have disabled as true.
					if !isDisabled {
						status.Failure("The IPPool %q with CIDR %q for remote endpoint"+
							" %q has disabled set to false", ipPool.GetName(), subnet, connection.Endpoint.CableName)

						failed = true

						continue
					}
				} else {
					status.Failure("Could not find any IPPool with CIDR %q for remote"+
						" endpoint %q", subnet, connection.Endpoint.CableName)

					failed = true
					continue
				}
			}
		}
	}

	return failed
}

func getSpecBool(pool unstructured.Unstructured, key string) (bool, error) {
	isDisabled, found, err := unstructured.NestedBool(pool.Object, "spec", key)
	if err != nil {
		return false, errors.Wrap(err, "error getting spec field")
	}

	if !found {
		message := fmt.Sprintf("%s status not found for IPPool %q", key, pool.GetName())
		return false, fmt.Errorf(message)
	}

	return isDisabled, nil
}
