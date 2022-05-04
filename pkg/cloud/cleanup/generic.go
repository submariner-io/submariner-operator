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
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/generic"
)

func newGenericCleanupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generic",
		Short: "Cleans up a cluster after Submariner uninstallation",
		Long:  "This command removes the labels from gateway nodes after Submariner uninstallation.",
		Run:   cleanupGenericCluster,
	}

	return cmd
}

func cleanupGenericCluster(cmd *cobra.Command, args []string) {
	err := generic.RunOnK8sCluster(
		*parentRestConfigProducer, cli.NewReporter(),
		func(gwDeployer api.GatewayDeployer, status reporter.Interface) error {
			return gwDeployer.Cleanup(status) // nolint:wrapcheck // No need to wrap here
		})

	exit.OnErrorWithMessage(err, "Failed to cleanup K8s cluster")
}
