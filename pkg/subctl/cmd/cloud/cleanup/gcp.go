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
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/cloud/cleanup"
	cloudgcp "github.com/submariner-io/submariner-operator/pkg/cloud/gcp"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/gcp"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
)

var gcpConfig cloudgcp.Config

// newGCPCleanupCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func newGCPCleanupCommand(restConfigProducer restconfig.Producer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gcp",
		Short: "Clean up a GCP cloud",
		Long:  "This command cleans up an installer-provisioned infrastructure (IPI) on GCP-based cloud after Submariner uninstallation.",
		Run: func(cmd *cobra.Command, args []string) {
			status := cli.NewReporter()

			var err error
			if gcpConfig.OcpMetadataFile != "" {
				gcpConfig.InfraID, gcpConfig.Region, gcpConfig.ProjectID, err = cloudgcp.ReadFromFile(gcpConfig.OcpMetadataFile)
				exit.OnErrorWithMessage(err, "Failed to read GCP Cluster information from OCP metadata file")
			} else {
				utils.ExpectFlag(infraIDFlag, gcpConfig.InfraID)
				utils.ExpectFlag(regionFlag, gcpConfig.Region)
				utils.ExpectFlag(projectIDFlag, gcpConfig.Region)
			}

			gcpConfig.GWInstanceType = ""
			gcpConfig.DedicatedGateway = false

			status.Start("Retrieving GCP credentials from your GCP configuration")

			creds, err := cloudgcp.GetCredentials(gcpConfig.CredentialsFile)
			exit.OnError(status.Error(err, "error retrieving GCP credentials"))

			err = cleanup.GCP(&restConfigProducer, &gcpConfig, creds, status)
			exit.OnError(err)
		},
	}

	gcp.AddGCPFlags(cmd, &gcpConfig)

	return cmd
}
