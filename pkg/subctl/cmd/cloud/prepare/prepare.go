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
)

var (
	nattPort         uint16
	natDiscoveryPort uint16
	vxlanPort        uint16
	metricsPort      uint16
	kubeConfig       *string
	kubeContext      *string
)

var (
	awsGWInstanceType string
	gcpGWInstanceType string
	gateways          int
	dedicatedGateway  bool
)

const DefaultNumGateways = 1

// NewCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func NewCommand(origKubeConfig, origKubeContext *string) *cobra.Command {
	kubeConfig = origKubeConfig
	kubeContext = origKubeContext
	cmd := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare the cloud",
		Long:  `This command prepares the cloud for Submariner installation.`,
	}

	cmd.PersistentFlags().Uint16Var(&nattPort, "natt-port", 4500, "IPSec NAT traversal port")
	cmd.PersistentFlags().Uint16Var(&natDiscoveryPort, "nat-discovery-port", 4490, "NAT discovery port")
	cmd.PersistentFlags().Uint16Var(&vxlanPort, "vxlan-port", 4800, "Internal VXLAN port")
	cmd.PersistentFlags().Uint16Var(&metricsPort, "metrics-port", 8080, "Metrics port")

	cmd.AddCommand(newAWSPrepareCommand())
	cmd.AddCommand(newGCPPrepareCommand())
	cmd.AddCommand(newGenericPrepareCommand())

	return cmd
}
