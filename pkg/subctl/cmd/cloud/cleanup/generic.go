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

package cleanup

import (
	"github.com/spf13/cobra"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/generic"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
)

func newGenericCleanupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generic",
		Short: "Clean up a k8s cluster",
		Long:  "This command cleans up a K8s cluster after Submariner uninstallation.",
		Run:   cleanupGenericCluster,
	}

	return cmd
}

func cleanupGenericCluster(cmd *cobra.Command, args []string) {
	err := generic.RunOnK8sCluster(*kubeConfig, *kubeContext,
		func(gwDeployer api.GatewayDeployer, reporter api.Reporter) error {
			return gwDeployer.Cleanup(reporter)
		})

	utils.ExitOnError("Failed to cleanup K8s cluster", err)
}
