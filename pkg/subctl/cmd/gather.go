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
	"k8s.io/client-go/kubernetes"

	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/gather"
	"github.com/submariner-io/submariner-operator/pkg/subctl/components"
)

var (
	gatherType   string
	gatherModule string
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

var gatherFuncs = map[string]func(*cli.Status, kubernetes.Interface, string, gather.GatherParams) error{
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
}

var gatherCmd = &cobra.Command{
	Use:   "gather <kubeConfig>",
	Short: "Gather troubleshooting data from a cluster",
	Long: fmt.Sprintf("This command gathers data from a submariner cluster for troubleshooting. The data gathered "+
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
	exitOnError("Error getting REST config", err)

	restconfig, err := config.ClientConfig()
	exitOnError("error getting rest config", err)

	clientSet, err := kubernetes.NewForConfig(restconfig)
	exitOnError("error getting clientset", err)

	rawconfig, err := config.RawConfig()
	exitOnError("error getting Raw config", err)

	clustername := getClusterNameFromContext(rawconfig, "")

	dirname := "submariner-" + time.Now().UTC().Format("20060102150405") // submariner-YYYYMMDDHHMMSS
	if _, err := os.Stat(dirname); os.IsNotExist(err) {
		err := os.MkdirAll(dirname, 0777)
		if err != nil {
			exitOnError(fmt.Sprintf("error creating directory %s", dirname), err)
		}
	}

	for module, ok := range gatherModuleFlags {
		if ok {
			for dataType, ok := range gatherTypeFlags {
				if ok {
					status := cli.NewStatus()
					status.Start(fmt.Sprintf("Gathering %s %s", module, dataType))
					status.End(cli.CheckForError(gatherFuncs[module](status, clientSet, dataType,
						gather.GatherParams{DirName: dirname, ClusterName: *clustername})))
				}
			}
		}
	}
}

func gatherConnectivity(status *cli.Status, clientSet kubernetes.Interface, dataType string, params gather.GatherParams) error {
	switch dataType {
	case Logs:
		err := gather.GatherGatewayPodLogs(clientSet, params)
		if err != nil {
			status.QueueFailureMessage(fmt.Sprintf("Failed to gather Gateway pod logs: %s", err))
		} else {
			status.QueueSuccessMessage("Successfully gathered Gateway pod logs")
		}
		err = gather.GatherRouteagentPodLogs(clientSet, params)
		if err != nil {
			status.QueueFailureMessage(fmt.Sprintf("Failed to gather Routeagent pod logs: %s", err))
		} else {
			status.QueueSuccessMessage("Successfully gathered Routeagent pod logs")
		}
	case Resources:
		status.QueueWarningMessage("Gather Connectivity Resources not implemented yet")
	default:
		return fmt.Errorf("unsupported data type %s", dataType)
	}
	return nil
}

func gatherDiscovery(status *cli.Status, clientSet kubernetes.Interface, dataType string, params gather.GatherParams) error {
	switch dataType {
	case Logs:
		status.QueueWarningMessage("Gather ServiceDiscovery Logs not implemented yet")
	case Resources:
		status.QueueWarningMessage("Gather ServiceDiscovery Resources not implemented yet")
	default:
		return fmt.Errorf("unsupported data type %s", dataType)
	}
	return nil
}

func gatherBroker(status *cli.Status, clientSet kubernetes.Interface, dataType string, params gather.GatherParams) error {
	switch dataType {
	case Logs:
		status.QueueWarningMessage("No logs to gather on Broker")
	case Resources:
		status.QueueWarningMessage("Gather Broker Resources not implemented yet")
	default:
		return fmt.Errorf("unsupported data type %s", dataType)
	}
	return nil
}

func gatherOperator(status *cli.Status, clientSet kubernetes.Interface, dataType string, params gather.GatherParams) error {
	switch dataType {
	case Logs:
		status.QueueWarningMessage("Gather Operator Logs not implemented yet")
	case Resources:
		status.QueueWarningMessage("Gather Operator Resources not implemented yet")
	default:
		return fmt.Errorf("unsupported data type %s", dataType)
	}
	return nil
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
