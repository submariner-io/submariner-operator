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
	"strings"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
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

var gatherFuncs = map[string]func(*cli.Status, string) error{
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

	for module, ok := range gatherModuleFlags {
		if ok {
			for dataType, ok := range gatherTypeFlags {
				if ok {
					status := cli.NewStatus()
					status.Start(fmt.Sprintf("Gathering %s %s...", module, dataType))
					status.End(cli.CheckForError(gatherFuncs[module](status, dataType)))
				}
			}
		}
	}
}

func gatherConnectivity(status *cli.Status, dataType string) error {
	switch dataType {
	case Logs:
		status.QueueWarningMessage("Gather Connectivity Logs not implemented yet")
	case Resources:
		status.QueueWarningMessage("Gather Connectivity Resources not implemented yet")
	}
	return nil
}

func gatherDiscovery(status *cli.Status, dataType string) error {
	switch dataType {
	case Logs:
		status.QueueWarningMessage("Gather ServiceDiscovery Logs not implemented yet")
	case Resources:
		status.QueueWarningMessage("Gather ServiceDiscovery Resources not implemented yet")
	}
	return nil
}

func gatherBroker(status *cli.Status, dataType string) error {
	if dataType == Resources {
		status.QueueWarningMessage("Gather Broker Resources not implemented yet")
	}
	return nil
}

func gatherOperator(status *cli.Status, dataType string) error {
	switch dataType {
	case Logs:
		status.QueueWarningMessage("Gather Operator Logs not implemented yet")
	case Resources:
		status.QueueWarningMessage("Gather Operator Resources not implemented yet")
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
