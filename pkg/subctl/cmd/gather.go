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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/gather"
	"github.com/submariner-io/submariner-operator/pkg/subctl/components"
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

var gatherFuncs = map[string]func(string, *gather.Info) bool{
	components.Connectivity:     gatherConnectivity,
	components.ServiceDiscovery: gatherDiscovery,
	components.Broker:           gatherBroker,
	components.Operator:         gatherOperator,
}

func init() {
	addKubeconfigFlag(gatherCmd)
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

	config := getClientConfig(kubeConfig, kubeContext)
	exitOnError("Error getting client config", err)

	restConfig, err := config.ClientConfig()
	exitOnError("Error getting REST config", err)

	rawConfig, err := config.RawConfig()
	exitOnError("Error getting raw config", err)

	clusterName := *getClusterNameFromContext(rawConfig, "")

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

	info := &gather.Info{
		RestConfig:  restConfig,
		ClusterName: clusterName,
		DirName:     directory,
	}

	info.ClientSet, err = kubernetes.NewForConfig(restConfig)
	exitOnError("Error creating k8s client set", err)

	info.DynClient, err = dynamic.NewForConfig(restConfig)
	exitOnError("Error creating dynamic client", err)

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

func gatherConnectivity(dataType string, info *gather.Info) bool {
	switch dataType {
	case Logs:
		err := gather.GatewayPodLogs(info)
		if err != nil {
			info.Status.QueueFailureMessage(fmt.Sprintf("Failed to gather Gateway pod logs: %s", err))
		}

		err = gather.RouteAgentPodLogs(info)
		if err != nil {
			info.Status.QueueFailureMessage(fmt.Sprintf("Failed to gather Route Agent pod logs: %s", err))
		}

		err = gather.GlobalnetPodLogs(info)
		if err != nil {
			info.Status.QueueFailureMessage(fmt.Sprintf("Failed to gather Globalnet pod logs: %s", err))
		}

		err = gather.NetworkPluginSyncerPodLogs(info)
		if err != nil {
			info.Status.QueueFailureMessage(fmt.Sprintf("Failed to gather NetworkPluginSyncer pod logs: %s", err))
		}
	case Resources:
		gather.Endpoints(info, SubmarinerNamespace)
		gather.Clusters(info, SubmarinerNamespace)
		gather.Gateways(info, SubmarinerNamespace)
	default:
		return false
	}

	return true
}

func gatherDiscovery(dataType string, info *gather.Info) bool {
	switch dataType {
	case Logs:
		info.Status.QueueWarningMessage("Gather ServiceDiscovery Logs not implemented yet")
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

func gatherBroker(dataType string, info *gather.Info) bool {
	switch dataType {
	case Logs:
		info.Status.QueueWarningMessage("No logs to gather on Broker")
	case Resources:
		_, _ = getBrokerRestConfig(info.RestConfig)

		info.Status.QueueWarningMessage("Gather Broker Resources not implemented yet")
	default:
		return false
	}

	return true
}

func gatherOperator(dataType string, info *gather.Info) bool {
	switch dataType {
	case Logs:
		info.Status.QueueWarningMessage("Gather Operator Logs not implemented yet")
	case Resources:
		gather.OperatorSubmariner(info, SubmarinerNamespace)
		gather.OperatorServiceDiscovery(info, SubmarinerNamespace)
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
