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
	"os"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/cloud/cleanup"
	cloudrhos "github.com/submariner-io/submariner-operator/pkg/cloud/rhos"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/rhos"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
)

var rhosConfig cloudrhos.Config

// newRHOSCleanupCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func newRHOSCleanupCommand(restConfigProducer restconfig.Producer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rhos",
		Short: "Clean up an RHOS cloud",
		Long: "This command cleans up an OpenShift installer-provisioned infrastructure (IPI) on RHOS-based" +
			" cloud after Submariner uninstallation.",
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

			rhosConfig.GWInstanceType = ""
			rhosConfig.DedicatedGateway = false

			err = cleanup.RHOS(&restConfigProducer, &rhosConfig, status)
			exit.OnError(err)
		},
	}

	rhos.AddRHOSFlags(cmd, &rhosConfig)

	return cmd
}
