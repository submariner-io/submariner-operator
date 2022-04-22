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
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/cmd/subctl/execute"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/internal/gather"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
)

var (
	options      gather.Options
	gatherType   string
	gatherModule string
)

func init() {
	restConfigProducer.AddKubeContextMultiFlag(gatherCmd, "")
	addGatherFlags(gatherCmd)
	AddToRootCommand(gatherCmd)
}

func addGatherFlags(gatherCmd *cobra.Command) {
	gatherCmd.Flags().StringVar(&gatherType, "type", strings.Join(getAllTypeKeys(), ","),
		"comma-separated list of data types to gather")
	gatherCmd.Flags().StringVar(&gatherModule, "module", strings.Join(getAllModuleKeys(), ","),
		"comma-separated list of components for which to gather data")
	gatherCmd.Flags().StringVar(&options.Directory, "dir", "",
		"the directory in which to store files. If not specified, a directory of the form \"submariner-<timestamp>\" "+
			"is created in the current directory")
	gatherCmd.Flags().BoolVar(&options.IncludeSensitiveData, "include-sensitive-data", false,
		"do not redact sensitive data such as credentials and security tokens")
}

var gatherCmd = &cobra.Command{
	Use:   "gather",
	Short: "Gather troubleshooting information from a cluster",
	Long: fmt.Sprintf("This command gathers information from a submariner cluster for troubleshooting. The information gathered "+
		"can be selected by component (%v) and type (%v). Default is to capture all data.",
		strings.Join(getAllModuleKeys(), ","), strings.Join(getAllTypeKeys(), ",")),
	Run: func(command *cobra.Command, args []string) {
		execute.OnMultiCluster(restConfigProducer, func(info *cluster.Info, status reporter.Interface) bool {
			options, err := checkGatherArguments(options)
			exit.OnErrorWithMessage(err, "Invalid arguments")

			return gather.Data(info, status, options)
		})
	},
}

func checkGatherArguments(options gather.Options) (gather.Options, error) {
	gatherTypeList := strings.Split(gatherType, ",")
	for _, arg := range gatherTypeList {
		if _, found := gather.TypeFlags[arg]; !found {
			return gather.Options{}, fmt.Errorf("%s is not a supported type", arg)
		}

		options.TypeFlags[arg] = true
	}

	gatherModuleList := strings.Split(gatherModule, ",")
	for _, arg := range gatherModuleList {
		if _, found := gather.ModuleFlags[arg]; !found {
			return gather.Options{}, fmt.Errorf("%s is not a supported module", arg)
		}

		options.ModuleFlags[arg] = true
	}

	return options, nil
}

func getAllTypeKeys() []string {
	keys := []string{}

	for k := range gather.TypeFlags {
		keys = append(keys, k)
	}

	return keys
}

func getAllModuleKeys() []string {
	keys := []string{}

	for k := range gather.ModuleFlags {
		keys = append(keys, k)
	}

	return keys
}
