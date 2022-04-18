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
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/cloud"
)

var (
	nattPort         uint16
	natDiscoveryPort uint16
	vxlanPort        uint16
	metricsPort      uint16
)

var (
	gcpGWInstanceType  string
	rhosGWInstanceType string
	gateways           int
	dedicatedGateway   bool
)

var (
	parentRestConfigProducer *restconfig.Producer
	ports                    cloud.Ports
)

const DefaultNumGateways = 1

// NewCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func NewCommand(restConfigProducer *restconfig.Producer) *cobra.Command {
	parentRestConfigProducer = restConfigProducer
	cmd := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare the cloud",
		Long:  `This command prepares the cloud for Submariner installation.`,
	}

	cmd.PersistentFlags().Uint16Var(&ports.Natt, "natt-port", 4500, "IPSec NAT traversal port")
	cmd.PersistentFlags().Uint16Var(&ports.NatDiscovery, "nat-discovery-port", 4490, "NAT discovery port")
	cmd.PersistentFlags().Uint16Var(&ports.Vxlan, "vxlan-port", 4800, "Internal VXLAN port")
	cmd.PersistentFlags().Uint16Var(&ports.Metrics, "metrics-port", 8080, "Metrics port")

	cmd.AddCommand(newAWSPrepareCommand(restConfigProducer, ports))
	cmd.AddCommand(newGCPPrepareCommand())
	cmd.AddCommand(newRHOSPrepareCommand())
	cmd.AddCommand(newGenericPrepareCommand())

	return cmd
}
