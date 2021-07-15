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

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	"github.com/submariner-io/submariner/pkg/routeagent_driver/constants"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

var supportedNetworkPlugins = []string{constants.NetworkPluginGeneric, constants.NetworkPluginCanalFlannel, constants.NetworkPluginWeaveNet,
	constants.NetworkPluginOpenShiftSDN, constants.NetworkPluginOVNKubernetes, constants.NetworkPluginCalico}

var (
	calicoGVR = schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "ippools",
	}
)

func init() {
	diagnoseCmd.AddCommand(&cobra.Command{
		Use:   "cni",
		Short: "Check the CNI network plugin",
		Long:  "This command checks if the detected CNI network plugin is supported by Submariner.",
		Run: func(command *cobra.Command, args []string) {
			cmd.ExecuteMultiCluster(checkCNIConfig)
		},
	})
}
func checkCNIConfig(cluster *cmd.Cluster) bool {
	status := cli.NewStatus()

	if cluster.Submariner == nil {
		status.Start(cmd.SubmMissingMessage)
		status.End(cli.Warning)
		return true
	}

	status.Start("Checking Submariner support for the CNI network plugin")

	isSupportedPlugin := false
	for _, np := range supportedNetworkPlugins {
		if cluster.Submariner.Status.NetworkPlugin == np {
			isSupportedPlugin = true
			break
		}
	}

	if !isSupportedPlugin {
		status.EndWithFailure("The detected CNI network plugin (%q) is not supported by Submariner."+
			" Supported network plugins: %v\n", cluster.Submariner.Status.NetworkPlugin, supportedNetworkPlugins)
		return false
	}

	status.EndWithSuccess("The detected CNI network plugin (%q) is supported", cluster.Submariner.Status.NetworkPlugin)

	return checkCalicoIPPoolsIfCalicoCNI(cluster)
}

func findCalicoConfigMap(clientSet kubernetes.Interface) (*v1.ConfigMap, error) {
	cmList, err := clientSet.CoreV1().ConfigMaps(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, cm := range cmList.Items {
		if cm.Name == "calico-config" {
			return &cm, nil
		}
	}
	return nil, nil
}

func checkCalicoIPPoolsIfCalicoCNI(info *cmd.Cluster) bool {
	status := cli.NewStatus()

	calicoConfig, err := findCalicoConfigMap(info.KubeClient)
	if err != nil {
		status.Start(fmt.Sprintf("Error trying to detect the Calico ConfigMap: %s", err))
		status.End(cli.Failure)
		return false
	}

	if calicoConfig == nil {
		return true
	}

	status.Start("Calico CNI detected, checking if the Submariner IPPool pre-requisites are configured.")

	gateways, err := info.GetGateways()
	if err != nil {
		status.EndWithFailure("Error retrieving Gateways: %v", err)
		return false
	}

	if len(gateways.Items) == 0 {
		status.EndWithWarning("There are no gateways detected on the cluster")
		return false
	}

	client := info.DynClient.Resource(calicoGVR)

	ippoolList, err := client.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		status.EndWithFailure("Error obtaining IPPools: %v", err)
		return false
	}

	if len(ippoolList.Items) < 1 {
		status.EndWithFailure("Could not find any IPPools in the cluster")
		return false
	}

	ippools := make(map[string]unstructured.Unstructured)
	for _, pool := range ippoolList.Items {
		cidr, found, err := unstructured.NestedString(pool.Object, "spec", "cidr")
		if err != nil {
			status.QueueFailureMessage(fmt.Sprintf("Error extracting field cidr from IPPool %q", pool.GetName()))
			continue
		}

		if !found {
			status.QueueFailureMessage(fmt.Sprintf("No CIDR found in IPPool %q", pool.GetName()))
			continue
		}
		ippools[cidr] = pool
	}

	for _, gateway := range gateways.Items {
		if gateway.Status.HAStatus != submv1.HAStatusActive {
			continue
		}

		for _, connection := range gateway.Status.Connections {
			for _, subnet := range connection.Endpoint.Subnets {
				ipPool, found := ippools[subnet]
				if found {
					isDisabled, err := getSpecBool(ipPool, "disabled")
					if err != nil {
						status.QueueFailureMessage(err.Error())
						continue
					}

					// When disabled is set to true, Calico IPAM will not assign addresses from this Pool.
					// The IPPools configured for Submariner remote CIDRs should have disabled as true.
					if !isDisabled {
						status.QueueFailureMessage(fmt.Sprintf("The IPPool %q with CIDR %q for remote endpoint"+
							" %q has disabled set to false", ipPool.GetName(), subnet, connection.Endpoint.CableName))
						continue
					}
				} else {
					status.QueueFailureMessage(fmt.Sprintf("Could not find any IPPool with CIDR %q for remote"+
						" endpoint %q", subnet, connection.Endpoint.CableName))
					continue
				}
			}
		}
	}

	result := status.ResultFromMessages()
	status.End(result)

	return result != cli.Failure
}

func getSpecBool(pool unstructured.Unstructured, key string) (bool, error) {
	isDisabled, found, err := unstructured.NestedBool(pool.Object, "spec", key)
	if err != nil {
		return false, err
	}

	if !found {
		message := fmt.Sprintf("%s status not found for IPPool %q", key, pool.GetName())
		return false, fmt.Errorf(message)
	}
	return isDisabled, nil
}
