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
	"strings"
	"time"

	"github.com/spf13/cobra"
	submarinerOp "github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/gather"
	"github.com/submariner-io/submariner-operator/pkg/subctl/components"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	gatherType   string
	gatherModule string
	directory    string
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

var submarinerCR *submarinerOp.Submariner

func init() {
	addKubeconfigFlag(gatherCmd)
	addKubecontextsFlag(gatherCmd)
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

	configs, err := getMultipleRestConfigs(kubeConfig, kubeContexts)
	exitOnError("Error getting REST configs", err)

	for _, config := range configs {
		gatherDataByCluster(config)
	}
}

func gatherDataByCluster(restConfig restConfig) {
	var err error
	clusterName := restConfig.clusterName

	if directory == "" {
		directory = "submariner-" + time.Now().UTC().Format("20060102150405") // submariner-YYYYMMDDHHMMSS
	}

	if _, err := os.Stat(directory); os.IsNotExist(err) {
		err := os.MkdirAll(directory, 0700)
		if err != nil {
			exitOnError(fmt.Sprintf("Error creating directory %q", directory), err)
		}
	}

	fmt.Printf("Gathering information from cluster %q\n", clusterName)

	info := gather.Info{
		RestConfig:  restConfig.config,
		ClusterName: clusterName,
		DirName:     directory,
	}

	info.DynClient, info.ClientSet, err = getClients(restConfig.config)
	exitOnError("Error getting client %s", err)

	resourceSubmariners := schema.GroupVersionResource{
		Group:    submarinerOp.SchemeGroupVersion.Group,
		Version:  submarinerOp.SchemeGroupVersion.Version,
		Resource: "submariners",
	}

	resourceSubm, err := info.DynClient.Resource(resourceSubmariners).Namespace(submarinerNamespace).Get("submariner", metav1.GetOptions{})
	if err == nil {
		var submariner submarinerOp.Submariner
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(resourceSubm.UnstructuredContent(), &submariner)
		if err != nil {
			exitOnError(fmt.Sprintf("Failed to get submariner resource %q", err), err)
		}
		submarinerCR = &submariner
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
	switch dataType {
	case Logs:
		gather.GatewayPodLogs(info)
		gather.RouteAgentPodLogs(info)
		gather.GlobalnetPodLogs(info)
		gather.NetworkPluginSyncerPodLogs(info)
	case Resources:
		if submarinerCR != nil {
			gather.CNIResources(info, submarinerCR.Status.NetworkPlugin)
			gather.CableDriverResources(info, submarinerCR.Spec.CableDriver)
		}
		gather.Endpoints(info, SubmarinerNamespace)
		gather.Clusters(info, SubmarinerNamespace)
		gather.Gateways(info, SubmarinerNamespace)
	default:
		return false
	}

	return true
}

func gatherDiscovery(dataType string, info gather.Info) bool {
	switch dataType {
	case Logs:
		gather.ServiceDiscoveryPodLogs(info)
		gather.CoreDNSPodLogs(info)
	case Resources:
		gather.ServiceExports(info, corev1.NamespaceAll)
		gather.ServiceImports(info, corev1.NamespaceAll)
		gather.EndpointSlices(info, corev1.NamespaceAll)
		gather.ConfigMapLighthouseDNS(info, SubmarinerNamespace)
		gather.ConfigMapCoreDNS(info, "kube-system")
	default:
		return false
	}

	return true
}

func gatherBroker(dataType string, info gather.Info) bool {
	switch dataType {
	case Resources:
		brokerRestConfig, brokerNamespace, err := getBrokerRestConfigAndNamespace(info.RestConfig)
		if err != nil {
			info.Status.QueueFailureMessage(fmt.Sprintf("Error getting the broker's rest config: %s", err))
			return true
		}

		info.DynClient, info.ClientSet, err = getClients(brokerRestConfig)
		if err != nil {
			info.Status.QueueFailureMessage(fmt.Sprintf("Error getting the broker client %s", err))
			return true
		}

		info.RestConfig = brokerRestConfig
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
