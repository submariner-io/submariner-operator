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

package prepare

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/cloud"
	"github.com/submariner-io/submariner-operator/pkg/cloud/prepare"
	cloudrhos "github.com/submariner-io/submariner-operator/pkg/cloud/rhos"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/rhos"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
)

var rhosConfig cloudrhos.Config

// newRHOSPrepareCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func newRHOSPrepareCommand(restConfigProducer *restconfig.Producer, ports *cloud.Ports) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rhos",
		Short: "Prepare an OpenShift RHOS cloud",
		Long:  "This command prepares an OpenShift installer-provisioned infrastructure (IPI) on RHOS cloud for Submariner installation.",
		Run: func(cmd *cobra.Command, args []string) {
			status := cli.NewReporter()

			var err error
			if config.OcpMetadataFile != "" {
				rhosConfig.InfraID, rhosConfig.ProjectID, err = cloudrhos.ReadFromFile(config.OcpMetadataFile)
				rhosConfig.Region = os.Getenv("OS_REGION_NAME")

				exit.OnErrorWithMessage(err, "Failed to read RHOS Cluster information from OCP metadata file")
			} else {
				utils.ExpectFlag(infraIDFlag, rhosConfig.InfraID)
				utils.ExpectFlag(regionFlag, rhosConfig.Region)
				utils.ExpectFlag(projectIDFlag, rhosConfig.ProjectID)
			}

			err = prepare.RHOS(restConfigProducer, ports, &rhosConfig, status)
			exit.OnError(err)
		},
	}

	rhos.AddRHOSFlags(cmd, &rhosConfig)
	cmd.Flags().IntVar(&rhosConfig.Gateways, "gateways", DefaultNumGateways,
		"Number of gateways to deploy")
	cmd.Flags().StringVar(&rhosConfig.GWInstanceType, "gateway-instance", "PnTAE.CPU_4_Memory_8192_Disk_50",
		"Type of gateway instance machine")
	cmd.Flags().BoolVar(&rhosConfig.DedicatedGateway, "dedicated-gateway", true,
		"Whether a dedicated gateway node has to be deployed")

	return cmd
}
