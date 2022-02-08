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

package subctl

import (
	"fmt"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/cmd/subctl/execute"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/internal/gather"
)

var options gather.Options

var gatherCmd = &cobra.Command{
	Use:   "gather <kubeConfig>",
	Short: "Gather troubleshooting information from a cluster",
	Long: fmt.Sprintf("This command gathers information from a submariner cluster for troubleshooting. The information gathered "+
		"can be selected by component (%v) and type (%v). Default is to capture all data.",
		strings.Join(getAllModuleKeys(), ","), strings.Join(getAllTypeKeys(), ",")),
	Run: func(command *cobra.Command, args []string) {
		err := checkGatherArguments(options)
		exit.OnErrorWithMessage(err, "Invalid arguments")

		if options.Directory == "" {
			options.Directory = "submariner-" + time.Now().UTC().Format("20060102150405") // submariner-YYYYMMDDHHMMSS
		}

		if _, err := os.Stat(options.Directory); os.IsNotExist(err) {
			err := os.MkdirAll(options.Directory, 0o700)
			if err != nil {
				exit.OnErrorWithMessage(err, fmt.Sprintf("Error creating directory %q: %s", options.Directory))
			}
		}

		execute.OnMultiCluster(restConfigProducer, func(clusterInfo *cluster.Info, status reporter.Interface) bool {
			return gather.Data(clusterInfo, status, options)
		})
	},
}

func addGatherFlags(gatherCmd *cobra.Command) {
	gatherCmd.Flags().StringVar(&options.Type, "type", strings.Join(getAllTypeKeys(), ","),
		"comma-separated list of data types to gather")
	gatherCmd.Flags().StringVar(&options.Module, "module", strings.Join(getAllModuleKeys(), ","),
		"comma-separated list of components for which to gather data")
	gatherCmd.Flags().StringVar(&options.Directory, "dir", "",
		"the directory in which to store files. If not specified, a directory of the form \"submariner-<timestamp>\" "+
			"is created in the current directory")
	gatherCmd.Flags().BoolVar(&options.IncludeSensitiveData, "include-sensitive-data", false,
		"do not redact sensitive data such as credentials and security tokens")
}

func init() {
	restConfigProducer.AddKubeContextMultiFlag(gatherCmd, "")
	addGatherFlags(gatherCmd)
	rootCmd.AddCommand(gatherCmd)
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

func checkGatherArguments(options gather.Options) error {
	TypeList := strings.Split(options.Type, ",")
	for _, arg := range TypeList {
		if _, found := gather.TypeFlags[arg]; !found {
			return fmt.Errorf("%s is not a supported type", arg)
		}

		gather.TypeFlags[arg] = true
	}

	gatherModuleList := strings.Split(options.Module, ",")
	for _, arg := range gatherModuleList {
		if _, found := gather.ModuleFlags[arg]; !found {
			return fmt.Errorf("%s is not a supported module", arg)
		}

		gather.ModuleFlags[arg] = true
	}

	return nil
}
