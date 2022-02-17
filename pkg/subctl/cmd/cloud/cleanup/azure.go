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
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/azure"
)

// newAzureCleanupCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func newAzureCleanupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "azure",
		Short: "Clean up an Azure cloud",
		Long: "This command cleans up an OpenShift installer-provisioned infrastructure (IPI) on Azure-based" +
			" cloud after Submariner uninstallation.",
		Run: cleanupAzure,
	}

	azure.AddAzureFlags(cmd)

	return cmd
}

func cleanupAzure(cmd *cobra.Command, args []string) {
	err := azure.RunOnAzure(*parentRestConfigProducer, "", cli.NewReporter(),
		// nolint:wrapcheck // No need to wrap errors here
		func(cloud api.Cloud, gwDeployer api.GatewayDeployer, status reporter.Interface) error {
			err := gwDeployer.Cleanup(status)
			if err != nil {
				return err
			}
			return cloud.CleanupAfterSubmariner(status)
		})
	exit.OnErrorWithMessage(err, "Failed to cleanup Azure cloud")
}
