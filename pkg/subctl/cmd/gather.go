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

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	subOperatorClientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/names"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/gather"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/subctl/components"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/brokercr"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	gatherType           string
	gatherModule         string
	directory            string
	includeSensitiveData bool
)

const (
	Logs      = "logs"
	Resources = "resources"
)

var gatherModuleFlags = map[string]bool{
	components.Connectivity:     false,
	components.ServiceDiscovery: false,
	components.Broker:           false,
	components.Operator:         false,
}

var gatherTypeFlags = map[string]bool{
	"logs":      false,
	"resources": false,
}

var gatherFuncs = map[string]func(string, gather.Info) bool{
	components.Connectivity:     gatherConnectivity,
	components.ServiceDiscovery: gatherDiscovery,
	components.Broker:           gatherBroker,
	components.Operator:         gatherOperator,
}

func init() {
	AddKubeContextMultiFlag(gatherCmd)
	addGatherFlags(gatherCmd)
	rootCmd.AddCommand(gatherCmd)
}

func addGatherFlags(gatherCmd *cobra.Command) {
	gatherCmd.Flags().StringVar(&gatherType, "type", strings.Join(getAllTypeKeys(), ","),
		"comma-separated list of data types to gather")
	gatherCmd.Flags().StringVar(&gatherModule, "module", strings.Join(getAllModuleKeys(), ","),
		"comma-separated list of components for which to gather data")
	gatherCmd.Flags().StringVar(&directory, "dir", "",
		"the directory in which to store files. If not specified, a directory of the form \"submariner-<timestamp>\" "+
			"is created in the current directory")
	gatherCmd.Flags().BoolVar(&includeSensitiveData, "include-sensitive-data", false,
		"do not redact sensitive data such as credentials and security tokens")
}

var gatherCmd = &cobra.Command{
	Use:   "gather <kubeConfig>",
	Short: "Gather troubleshooting information from a cluster",
	Long: fmt.Sprintf("This command gathers information from a submariner cluster for troubleshooting. The information gathered "+
		"can be selected by component (%v) and type (%v). Default is to capture all data.",
		strings.Join(getAllModuleKeys(), ","), strings.Join(getAllTypeKeys(), ",")),
	Run: func(cmd *cobra.Command, args []string) {
		gatherData()
	},
}

func gatherData() {
	err := checkGatherArguments()
	exitOnError("Invalid arguments", err)

	configs, err := restconfig.ForClusters(kubeConfig, kubeContexts)
	exitOnError("Error getting REST configs", err)

	if directory == "" {
		directory = "submariner-" + time.Now().UTC().Format("20060102150405") // submariner-YYYYMMDDHHMMSS
	}

	if _, err := os.Stat(directory); os.IsNotExist(err) {
		err := os.MkdirAll(directory, 0700)
		if err != nil {
			exitOnError(fmt.Sprintf("Error creating directory %q", directory), err)
		}
	}

	for _, config := range configs {
		gatherDataByCluster(config, directory)
	}

	fmt.Printf("Files are stored under directory %q\n", directory)
}

func gatherDataByCluster(restConfig restconfig.RestConfig, directory string) {
	var err error
	clusterName := restConfig.ClusterName

	fmt.Printf("Gathering information from cluster %q\n", clusterName)

	info := gather.Info{
		RestConfig:           restConfig.Config,
		ClusterName:          clusterName,
		DirName:              directory,
		IncludeSensitiveData: includeSensitiveData,
	}

	info.DynClient, info.ClientSet, err = restconfig.Clients(restConfig.Config)
	if err != nil {
		fmt.Printf("Error getting client: %s\n", err)
		return
	}

	submarinerClient, err := subOperatorClientset.NewForConfig(restConfig.Config)
	if err != nil {
		fmt.Printf("Error getting Submariner client: %s\n", err)
		return
	}

	info.Submariner, err = submarinerClient.SubmarinerV1alpha1().Submariners(OperatorNamespace).
		Get(context.TODO(), submarinercr.SubmarinerName, metav1.GetOptions{})
	if err != nil {
		info.Submariner = nil
		if !apierrors.IsNotFound(err) {
			fmt.Printf("Error getting Submariner resource: %s\n", err)
			return
		}
	}

	info.ServiceDiscovery, err = submarinerClient.SubmarinerV1alpha1().ServiceDiscoveries(OperatorNamespace).
		Get(context.TODO(), names.ServiceDiscoveryCrName, metav1.GetOptions{})
	if err != nil {
		info.ServiceDiscovery = nil
		if !apierrors.IsNotFound(err) {
			fmt.Printf("Error getting ServiceDiscovery resource: %s\n", err)
			return
		}
	}

	for module, ok := range gatherModuleFlags {
		if ok {
			for dataType, ok := range gatherTypeFlags {
				if ok {
					info.Status = cli.NewStatus()
					info.Status.Start(fmt.Sprintf("Gathering %s %s", module, dataType))

					if gatherFuncs[module](dataType, info) {
						info.Status.End(info.Status.ResultFromMessages())
					}
				}
			}
		}
	}
}

func gatherConnectivity(dataType string, info gather.Info) bool {
	if info.Submariner == nil {
		info.Status.QueueWarningMessage("The Submariner connectivity components are not installed")
		return false
	}

	switch dataType {
	case Logs:
		gather.GatewayPodLogs(info)
		gather.RouteAgentPodLogs(info)
		gather.GlobalnetPodLogs(info)
		gather.NetworkPluginSyncerPodLogs(info)
	case Resources:
		gather.CNIResources(info, info.Submariner.Status.NetworkPlugin)
		gather.CableDriverResources(info, info.Submariner.Spec.CableDriver)
		gather.OVNResources(info, info.Submariner.Status.NetworkPlugin)
		gather.Endpoints(info, SubmarinerNamespace)
		gather.Clusters(info, SubmarinerNamespace)
		gather.Gateways(info, SubmarinerNamespace)
	default:
		return false
	}

	return true
}

func gatherDiscovery(dataType string, info gather.Info) bool {
	if info.ServiceDiscovery == nil {
		info.Status.QueueWarningMessage("The Submariner service discovery components are not installed")
		return false
	}

	switch dataType {
	case Logs:
		gather.ServiceDiscoveryPodLogs(info)
		gather.CoreDNSPodLogs(info)
	case Resources:
		gather.ServiceExports(info, corev1.NamespaceAll)
		gather.ServiceImports(info, corev1.NamespaceAll)
		gather.EndpointSlices(info, corev1.NamespaceAll)
		gather.ConfigMapLighthouseDNS(info, SubmarinerNamespace)
		gather.ConfigMapCoreDNS(info)
	default:
		return false
	}

	return true
}

func gatherBroker(dataType string, info gather.Info) bool {
	switch dataType {
	case Resources:
		brokerRestConfig, brokerNamespace, err := restconfig.ForBroker(info.Submariner, info.ServiceDiscovery)
		if err != nil {
			info.Status.QueueFailureMessage(fmt.Sprintf("Error getting the broker's rest config: %s", err))
			return true
		}

		if brokerRestConfig != nil {
			info.RestConfig = brokerRestConfig
			info.DynClient, info.ClientSet, err = restconfig.Clients(brokerRestConfig)
			if err != nil {
				info.Status.QueueFailureMessage(fmt.Sprintf("Error getting the broker client: %s", err))
				return true
			}
		} else {
			submarinerClient, err := subOperatorClientset.NewForConfig(info.RestConfig)
			if err != nil {
				info.Status.QueueFailureMessage(fmt.Sprintf("Error getting the Submariner client: %s", err))
				return true
			}

			_, err = submarinerClient.SubmarinerV1alpha1().Brokers(OperatorNamespace).Get(
				context.TODO(), brokercr.BrokerName, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return false
			}

			if err != nil {
				info.Status.QueueFailureMessage(fmt.Sprintf("Error getting the Broker resource: %s", err))
				return true
			}

			brokerNamespace = metav1.NamespaceAll
		}

		info.ClusterName = "broker"

		// The broker's ClusterRole used by member clusters only allows the below resources to be queried
		gather.Endpoints(info, brokerNamespace)
		gather.Clusters(info, brokerNamespace)
		gather.EndpointSlices(info, brokerNamespace)
		gather.ServiceImports(info, brokerNamespace)
	default:
		return false
	}

	return true
}

func gatherOperator(dataType string, info gather.Info) bool {
	switch dataType {
	case Logs:
		gather.SubmarinerOperatorPodLogs(info)
	case Resources:
		gather.Submariners(info, SubmarinerNamespace)
		gather.ServiceDiscoveries(info, SubmarinerNamespace)
		gather.SubmarinerOperatorDeployment(info, SubmarinerNamespace)
		gather.GatewayDaemonSet(info, SubmarinerNamespace)
		gather.RouteAgentDaemonSet(info, SubmarinerNamespace)
		gather.GlobalnetDaemonSet(info, SubmarinerNamespace)
		gather.NetworkPluginSyncerDeployment(info, SubmarinerNamespace)
		gather.LighthouseAgentDeployment(info, SubmarinerNamespace)
		gather.LighthouseCoreDNSDeployment(info, SubmarinerNamespace)
	default:
		return false
	}

	return true
}

func checkGatherArguments() error {
	gatherTypeList := strings.Split(gatherType, ",")
	for _, arg := range gatherTypeList {
		if _, found := gatherTypeFlags[arg]; !found {
			return fmt.Errorf("%s is not a supported type", arg)
		}
		gatherTypeFlags[arg] = true
	}

	gatherModuleList := strings.Split(gatherModule, ",")
	for _, arg := range gatherModuleList {
		if _, found := gatherModuleFlags[arg]; !found {
			return fmt.Errorf("%s is not a supported module", arg)
		}
		gatherModuleFlags[arg] = true
	}

	return nil
}

func getAllTypeKeys() []string {
	keys := []string{}

	for k := range gatherTypeFlags {
		keys = append(keys, k)
	}

	return keys
}

func getAllModuleKeys() []string {
	keys := []string{}

	for k := range gatherModuleFlags {
		keys = append(keys, k)
	}
	return keys
}
