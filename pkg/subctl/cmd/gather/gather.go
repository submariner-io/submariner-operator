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

package gather

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/component"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	subOperatorClientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/names"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
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
	restConfigProducer   = restconfig.NewProducer()
)

const (
	Logs      = "logs"
	Resources = "resources"
)

var gatherModuleFlags = map[string]bool{
	component.Connectivity:     false,
	component.ServiceDiscovery: false,
	component.Broker:           false,
	component.Operator:         false,
}

var gatherTypeFlags = map[string]bool{
	"logs":      false,
	"resources": false,
}

var gatherFuncs = map[string]func(string, Info) bool{
	component.Connectivity:     gatherConnectivity,
	component.ServiceDiscovery: gatherDiscovery,
	component.Broker:           gatherBroker,
	component.Operator:         gatherOperator,
}

func init() {
	restConfigProducer.AddKubeContextMultiFlag(gatherCmd, "")
	addGatherFlags(gatherCmd)
	cmd.AddToRootCommand(gatherCmd)
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
	Run: func(command *cobra.Command, args []string) {
		cmd.ExecuteMultiCluster(gatherData)
	},
}

func gatherData(cluster *cmd.Cluster) bool {
	err := checkGatherArguments()
	utils.ExitOnError("Invalid arguments", err)

	if directory == "" {
		directory = "submariner-" + time.Now().UTC().Format("20060102150405") // submariner-YYYYMMDDHHMMSS
	}

	if _, err := os.Stat(directory); os.IsNotExist(err) {
		err := os.MkdirAll(directory, 0o700)
		if err != nil {
			utils.ExitOnError(fmt.Sprintf("Error creating directory %q", directory), err)
		}
	}

	gatherDataByCluster(cluster, directory)

	fmt.Printf("Files are stored under directory %q\n", directory)

	return true
}

func gatherDataByCluster(cluster *cmd.Cluster, directory string) {
	var err error
	clusterName := cluster.Name

	fmt.Printf("Gathering information from cluster %q\n", clusterName)

	info := Info{
		RestConfig:           cluster.Config,
		ClusterName:          clusterName,
		DirName:              directory,
		IncludeSensitiveData: includeSensitiveData,
		Summary:              &Summary{},
	}

	info.DynClient = cluster.DynClient
	info.ClientSet = cluster.KubeClient

	submarinerClient, err := subOperatorClientset.NewForConfig(cluster.Config)
	if err != nil {
		fmt.Printf("Error getting Submariner client: %s\n", err)
		return
	}

	info.Submariner, err = submarinerClient.SubmarinerV1alpha1().Submariners(cmd.OperatorNamespace).
		Get(context.TODO(), submarinercr.SubmarinerName, metav1.GetOptions{})
	if err != nil {
		info.Submariner = nil

		if !apierrors.IsNotFound(err) {
			fmt.Printf("Error getting Submariner resource: %s\n", err)
			return
		}
	}

	info.ServiceDiscovery, err = submarinerClient.SubmarinerV1alpha1().ServiceDiscoveries(cmd.OperatorNamespace).
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
						info.Status.EndWith(info.Status.ResultFromMessages())
					}
				}
			}
		}
	}

	gatherClusterSummary(&info)
}

// nolint:gocritic // hugeParam: info - purposely passed by value.
func gatherConnectivity(dataType string, info Info) bool {
	if info.Submariner == nil {
		info.Status.QueueWarningMessage("The Submariner connectivity components are not installed")
		return true
	}

	switch dataType {
	case Logs:
		gatherGatewayPodLogs(&info)
		gatherRouteAgentPodLogs(&info)
		gatherGlobalnetPodLogs(&info)
		gatherNetworkPluginSyncerPodLogs(&info)
	case Resources:
		gatherCNIResources(&info, info.Submariner.Status.NetworkPlugin)
		gatherCableDriverResources(&info, info.Submariner.Spec.CableDriver)
		gatherOVNResources(&info, info.Submariner.Status.NetworkPlugin)
		gatherEndpoints(&info, cmd.SubmarinerNamespace)
		gatherClusters(&info, cmd.SubmarinerNamespace)
		gatherGateways(&info, cmd.SubmarinerNamespace)
		gatherClusterGlobalEgressIPs(&info)
		gatherGlobalEgressIPs(&info)
		gatherGlobalIngressIPs(&info)
	default:
		return false
	}

	return true
}

// nolint:gocritic // hugeParam: info - purposely passed by value.
func gatherDiscovery(dataType string, info Info) bool {
	if info.ServiceDiscovery == nil {
		info.Status.QueueWarningMessage("The Submariner service discovery components are not installed")
		return true
	}

	switch dataType {
	case Logs:
		gatherServiceDiscoveryPodLogs(&info)
		gatherCoreDNSPodLogs(&info)
	case Resources:
		gatherServiceExports(&info, corev1.NamespaceAll)
		gatherServiceImports(&info, corev1.NamespaceAll)
		gatherEndpointSlices(&info, corev1.NamespaceAll)
		gatherConfigMapLighthouseDNS(&info, cmd.SubmarinerNamespace)
		gatherConfigMapCoreDNS(&info)
	default:
		return false
	}

	return true
}

// nolint:gocritic // hugeParam: info - purposely passed by value.
func gatherBroker(dataType string, info Info) bool {
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

			_, err = submarinerClient.SubmarinerV1alpha1().Brokers(cmd.OperatorNamespace).Get(
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
		gatherEndpoints(&info, brokerNamespace)
		gatherClusters(&info, brokerNamespace)
		gatherEndpointSlices(&info, brokerNamespace)
		gatherServiceImports(&info, brokerNamespace)
	default:
		return false
	}

	return true
}

// nolint:gocritic // hugeParam: info - purposely passed by value.
func gatherOperator(dataType string, info Info) bool {
	switch dataType {
	case Logs:
		gatherSubmarinerOperatorPodLogs(&info)
	case Resources:
		gatherSubmariners(&info, cmd.SubmarinerNamespace)
		gatherServiceDiscoveries(&info, cmd.SubmarinerNamespace)
		gatherSubmarinerOperatorDeployment(&info, cmd.SubmarinerNamespace)
		gatherGatewayDaemonSet(&info, cmd.SubmarinerNamespace)
		gatherRouteAgentDaemonSet(&info, cmd.SubmarinerNamespace)
		gatherGlobalnetDaemonSet(&info, cmd.SubmarinerNamespace)
		gatherNetworkPluginSyncerDeployment(&info, cmd.SubmarinerNamespace)
		gatherLighthouseAgentDeployment(&info, cmd.SubmarinerNamespace)
		gatherLighthouseCoreDNSDeployment(&info, cmd.SubmarinerNamespace)
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
