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
)

var (
	gatherType   string
	gatherModule string
)

var gatherModuleFlags = map[string]bool{
	"connectivity": false,
	"discovery":    false,
	"broker":       false,
	"operator":     false,
}

var gatherTypeFlags = map[string]bool{
	"logs":      false,
	"resources": false,
}

func init() {
	addKubeconfigFlag(gatherCmd)
	addGatherFlags(gatherCmd)
	rootCmd.AddCommand(gatherCmd)
}

func addGatherFlags(gatherCmd *cobra.Command) {
	gatherCmd.Flags().StringVar(&gatherType, "type", strings.Join(getAllTypeKeys(), ","), "comma-separated list of data types to gather")
	gatherCmd.Flags().StringVar(&gatherModule, "module", strings.Join(getAllModuleKeys(), ","), "comma-separated list of components for which to gather data")
}

var gatherCmd = &cobra.Command{
	Use:   "gather <kubeConfig1>",
	Short: "Gather troubleshooting data from a cluster",
	Long: fmt.Sprintf("This command gathers data from a submariner cluster for troubleshooting. Data gathered" +
		"can be selected by component (%v) and type (%v). Default is to capture all data.", strings.Join(getAllModuleKeys(), ","), strings.Join(getAllTypeKeys(), ",")),
	Run: func(cmd *cobra.Command, args []string) {
		gatherData()
	},
}

func gatherData() {
	err := checkGatherArguments()
	exitOnError("Invalid arguments", err)

	fmt.Printf("Gathering following data for module(s): ")
	for module, ok := range gatherModuleFlags {
		if ok {
			fmt.Printf("%s ", module)
		}
	}
	for dataType, ok := range gatherTypeFlags {
		if ok {
			fmt.Printf("\n  * %s ", dataType)
		}
	}
	fmt.Println()
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
