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
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/pkg/cloud/cleanup"
	"github.com/submariner-io/submariner-operator/pkg/cloud/gcp"
	"github.com/submariner-io/submariner-operator/pkg/cloud/prepare"
)

var (
	gcpConfig gcp.Config

	gcpPrepareCmd = &cobra.Command{
		Use:     "gcp",
		Short:   "Prepare an OpenShift GCP cloud",
		Long:    "This command prepares an OpenShift installer-provisioned infrastructure (IPI) on GCP cloud for Submariner installation.",
		PreRunE: checkGCPFlags,
		Run: func(cmd *cobra.Command, args []string) {
			err := prepare.GCP(&restConfigProducer, &cloudPorts, &gcpConfig, cli.NewReporter())
			exit.OnError(err)
		},
	}

	gcpCleanupCmd = &cobra.Command{
		Use:     "gcp",
		Short:   "Clean up a GCP cloud",
		Long:    "This command cleans up an installer-provisioned infrastructure (IPI) on GCP-based cloud after Submariner uninstallation.",
		PreRunE: checkGCPFlags,
		Run: func(cmd *cobra.Command, args []string) {
			err := cleanup.GCP(&restConfigProducer, &gcpConfig, cli.NewReporter())
			exit.OnError(err)
		},
	}
)

func init() {
	addGCPGeneralFlags := func(command *cobra.Command) {
		command.Flags().StringVar(&gcpConfig.InfraID, infraIDFlag, "", "GCP infra ID")
		command.Flags().StringVar(&gcpConfig.Region, regionFlag, "", "GCP region")
		command.Flags().StringVar(&gcpConfig.ProjectID, projectIDFlag, "", "GCP project ID")
		command.Flags().StringVar(&gcpConfig.OcpMetadataFile, "ocp-metadata", "",
			"OCP metadata.json file (or the directory containing it) from which to read the GCP infra ID "+
				"and region from (takes precedence over the specific flags)")

		dirname, err := os.UserHomeDir()
		if err != nil {
			exit.OnErrorWithMessage(err, "failed to find home directory")
		}

		defaultCredentials := filepath.FromSlash(fmt.Sprintf("%s/.gcp/osServiceAccount.json", dirname))
		command.Flags().StringVar(&gcpConfig.CredentialsFile, "credentials", defaultCredentials, "GCP credentials configuration file")
	}

	addGCPGeneralFlags(gcpPrepareCmd)
	gcpPrepareCmd.Flags().StringVar(&gcpConfig.GWInstanceType, "gateway-instance", "n1-standard-4", "Type of gateway instance machine")
	gcpPrepareCmd.Flags().IntVar(&gcpConfig.Gateways, "gateways", defaultNumGateways,
		"Number of gateways to deploy")
	gcpPrepareCmd.Flags().BoolVar(&gcpConfig.DedicatedGateway, "dedicated-gateway", true,
		"Whether a dedicated gateway node has to be deployed")

	cloudPrepareCmd.AddCommand(gcpPrepareCmd)

	addGCPGeneralFlags(gcpCleanupCmd)
	cloudCleanupCmd.AddCommand(gcpCleanupCmd)
}

func checkGCPFlags(cmd *cobra.Command, args []string) error {
	if gcpConfig.OcpMetadataFile == "" {
		expectFlag(infraIDFlag, gcpConfig.InfraID)
		expectFlag(regionFlag, gcpConfig.Region)
		expectFlag(projectIDFlag, gcpConfig.ProjectID)
	}

	return nil
}
