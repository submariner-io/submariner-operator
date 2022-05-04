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
	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/rhos"
)

// newRHOSPrepareCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func newRHOSPrepareCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rhos",
		Short: "Prepare an OpenShift RHOS cloud",
		Long:  "This command prepares an OpenShift installer-provisioned infrastructure (IPI) on RHOS cloud for Submariner installation.",
		Run:   prepareRHOS,
	}

	rhos.AddRHOSFlags(cmd)
	cmd.Flags().IntVar(&gateways, "gateways", DefaultNumGateways,
		"Number of gateways to deploy")
	cmd.Flags().StringVar(&rhosGWInstanceType, "gateway-instance", "PnTAE.CPU_4_Memory_8192_Disk_50", "Type of gateway instance machine")
	cmd.Flags().BoolVar(&dedicatedGateway, "dedicated-gateway", true,
		"Whether a dedicated gateway node has to be deployed")

	return cmd
}
