/*
© 2021 Red Hat, Inc. and others.

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
package cmd

import (
	"fmt"
	"os"

	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

var supportedNetworkPlugins = []string{"generic", "canal-flannel", "weave-net", "OpenShiftSDN", "OVNKubernetes"}

var validateCniCmd = &cobra.Command{
	Use:   "cni",
	Short: "Check the CNI network plugin",
	Long:  "This command checks if the detected CNI network plugin is supported by Submariner.",
	Run:   validateCniConfig,
}

var (
	calicoGVR = schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "ippools",
	}
)

func init() {
	validateCmd.AddCommand(validateCniCmd)
}

func validateCniConfig(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContexts)
	exitOnError("Error getting REST config for cluster", err)

	validationStatus := true

	for _, item := range configs {
		status.Start(fmt.Sprintf("Retrieving Submariner resource from %q", item.clusterName))
		submariner := getSubmarinerResource(item.config)
		if submariner == nil {
			status.QueueWarningMessage(submMissingMessage)
			status.End(cli.Success)
			continue
		}
		status.End(cli.Success)
		if !validateCNIInCluster(item.config, item.clusterName, submariner) {
			validationStatus = false
		}
	}
	if !validationStatus {
		os.Exit(1)
	}
}

func validateCNIInCluster(config *rest.Config, clusterName string, submariner *v1alpha1.Submariner) bool {
	message := fmt.Sprintf("Checking Submariner support for the CNI network"+
		" plugin in cluster %q", clusterName)
	status.Start(message)

	isSupportedPlugin := false
	for _, np := range supportedNetworkPlugins {
		if submariner.Status.NetworkPlugin == np {
			isSupportedPlugin = true
			break
		}
	}

	if !isSupportedPlugin {
		message := fmt.Sprintf("The detected CNI network plugin (%q) is not supported by Submariner."+
			" Supported network plugins: %v\n", submariner.Status.NetworkPlugin, supportedNetworkPlugins)
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return false
	}

	message = fmt.Sprintf("The detected CNI network plugin (%q) is supported by Submariner.",
		submariner.Status.NetworkPlugin)
	status.QueueSuccessMessage(message)
	status.End(cli.Success)

	calicoCNIStatus := validateCalicoIPPoolsIfCalicoCNI(config)
	return calicoCNIStatus
}

func findCalicoConfigMap(clientSet kubernetes.Interface) (*v1.ConfigMap, error) {
	cmList, err := clientSet.CoreV1().ConfigMaps(metav1.NamespaceAll).List(metav1.ListOptions{})
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

func validateCalicoIPPoolsIfCalicoCNI(config *rest.Config) bool {
	dynClient, clientSet, err := getClients(config)
	if err != nil {
		message := fmt.Sprintf("Error creating client - unable to detect the Calico CNI: %s", err)
		status.Start(message)
		status.End(cli.Failure)
		return false
	}

	calicoConfig, err := findCalicoConfigMap(clientSet)
	if err != nil {
		message := fmt.Sprintf("Error trying to detect the Calico ConfigMap: %s", err)
		status.Start(message)
		status.End(cli.Failure)
		return false
	}

	if calicoConfig == nil {
		return true
	}

	message := "Calico CNI detected, verifying if the Submariner IPPool pre-requisites are configured."
	status.Start(message)

	gateways := getGatewaysResource(config)
	if gateways == nil {
		message = "There are no gateways detected on the cluster"
		status.QueueWarningMessage(message)
		status.End(cli.Warning)
		return false
	}

	client := dynClient.Resource(calicoGVR)

	ippoolList, err := client.List(metav1.ListOptions{})
	if err != nil {
		message := fmt.Sprintf("Error obtaining IPPools: %v", err)
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return false
	}

	if len(ippoolList.Items) < 1 {
		message := "Could not find any IPPools in the cluster"
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return false
	}

	ippools := make(map[string]unstructured.Unstructured)
	for _, pool := range ippoolList.Items {
		cidr, found, err := unstructured.NestedString(pool.Object, "spec", "cidr")
		if err != nil {
			message := fmt.Sprintf("Error extracting field cidr from IPPool %q", pool.GetName())
			status.QueueFailureMessage(message)
			continue
		}

		if !found {
			message := fmt.Sprintf("No CIDR found in IPPool %q", pool.GetName())
			status.QueueFailureMessage(message)
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
