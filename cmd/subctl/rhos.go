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
	"github.com/submariner-io/submariner-operator/pkg/cloud/rhos"
)

var (
	rhosConfig rhos.Config

	rhosPrepareCmd = &cobra.Command{
		Use:     "rhos",
		Short:   "Prepare an OpenShift RHOS cloud",
		Long:    "This command prepares an OpenShift installer-provisioned infrastructure (IPI) on RHOS cloud for Submariner installation.",
		PreRunE: checkRHOSFlags,
		Run: func(cmd *cobra.Command, args []string) {
			err := prepare.RHOS(&restConfigProducer, &cloudPorts, &rhosConfig, cli.NewReporter())
			exit.OnError(err)
		},
	}

	rhosCleanupCmd = &cobra.Command{
		Use:   "rhos",
		Short: "Clean up an RHOS cloud",
		Long: "This command cleans up an OpenShift installer-provisioned infrastructure (IPI) on RHOS-based" +
			" cloud after Submariner uninstallation.",
		PreRunE: checkRHOSFlags,
		Run: func(cmd *cobra.Command, args []string) {
			err := cleanup.RHOS(&restConfigProducer, &rhosConfig, cli.NewReporter())
			exit.OnError(err)
		},
	}
)

func init() {
	addGeneralRHOSFlags := func(command *cobra.Command) {
		command.Flags().StringVar(&rhosConfig.InfraID, infraIDFlag, "", "RHOS infra ID")
		command.Flags().StringVar(&rhosConfig.Region, regionFlag, "", "RHOS region")
		command.Flags().StringVar(&rhosConfig.ProjectID, projectIDFlag, "", "RHOS project ID")
		command.Flags().StringVar(&rhosConfig.OcpMetadataFile, "ocp-metadata", "",
			"OCP metadata.json file (or the directory containing it) from which to read the RHOS infra ID "+
				"and region from (takes precedence over the specific flags)")
		command.Flags().StringVar(&rhosConfig.CloudEntry, cloudEntryFlag, "", "the cloud entry to use")
	}

	addGeneralRHOSFlags(rhosPrepareCmd)
	rhosPrepareCmd.Flags().IntVar(&rhosConfig.Gateways, "gateways", defaultNumGateways,
		"Number of gateways to deploy")
	rhosPrepareCmd.Flags().StringVar(&rhosConfig.GWInstanceType, "gateway-instance", "PnTAE.CPU_4_Memory_8192_Disk_50",
		"Type of gateway instance machine")
	rhosPrepareCmd.Flags().BoolVar(&rhosConfig.DedicatedGateway, "dedicated-gateway", true,
		"Whether a dedicated gateway node has to be deployed")

	cloudPrepareCmd.AddCommand(rhosPrepareCmd)

	addGeneralRHOSFlags(rhosCleanupCmd)
	cloudCleanupCmd.AddCommand(rhosCleanupCmd)
}

func checkRHOSFlags(cmd *cobra.Command, args []string) error {
	if rhosConfig.OcpMetadataFile == "" {
		expectFlag(infraIDFlag, rhosConfig.InfraID)
		expectFlag(regionFlag, rhosConfig.Region)
		expectFlag(projectIDFlag, rhosConfig.ProjectID)
	}

	return nil
}
