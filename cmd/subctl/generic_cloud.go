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
	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/pkg/cloud/cleanup"
	"github.com/submariner-io/submariner-operator/pkg/cloud/prepare"
)

var (
	genericCloudConfig struct {
		gateways int
	}

	genericPrepareCmd = &cobra.Command{
		Use:   "generic",
		Short: "Prepares a generic cluster for Submariner",
		Long:  "This command labels the required number of gateway nodes for Submariner installation.",
		Run: func(cmd *cobra.Command, args []string) {
			exit.OnError(prepare.GenericCluster(&restConfigProducer, genericCloudConfig.gateways, cli.NewReporter()))
		},
	}

	genericCleanupCmd = &cobra.Command{
		Use:   "generic",
		Short: "Cleans up a cluster after Submariner uninstallation",
		Long:  "This command removes the labels from gateway nodes after Submariner uninstallation.",
		Run: func(cmd *cobra.Command, args []string) {
			exit.OnError(cleanup.GenericCluster(&restConfigProducer, cli.NewReporter()))
		},
	}
)

func init() {
	genericPrepareCmd.Flags().IntVar(&genericCloudConfig.gateways, "gateways", defaultNumGateways, "Number of gateways to deploy")
	cloudPrepareCmd.AddCommand(genericPrepareCmd)

	cloudCleanupCmd.AddCommand(genericCleanupCmd)
}
