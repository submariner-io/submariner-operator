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
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	subctlcmd "github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
)

var (
	nattPort         uint16
	natDiscoveryPort uint16
	vxlanPort        uint16
	metricsPort      []uint16
)

var (
	awsGWInstanceType  string
	gcpGWInstanceType  string
	rhosGWInstanceType string
	gateways           int
	dedicatedGateway   bool
)

var parentRestConfigProducer *restconfig.Producer

const DefaultNumGateways = 1

// NewCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func NewCommand(restConfigProducer *restconfig.Producer) *cobra.Command {
	parentRestConfigProducer = restConfigProducer
	cmd := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare the cloud",
		Long:  `This command prepares the cloud for Submariner installation.`,
	}

	cmd.PersistentFlags().Uint16Var(&nattPort, "natt-port", 4500, "IPSec NAT traversal port")
	cmd.PersistentFlags().Uint16Var(&natDiscoveryPort, "nat-discovery-port", 4490, "NAT discovery port")
	cmd.PersistentFlags().Uint16Var(&vxlanPort, "vxlan-port", 4800, "Internal VXLAN port")

	metricsPort = append(metricsPort, 8080, 8081)

	cmd.PersistentFlags().Var(&subctlcmd.Uint16Slice{Value: &metricsPort}, "metrics-ports", "Metrics ports")

	cmd.PersistentFlags().Var(&metricsAliasType{}, "metrics-port", "Metrics port")
	_ = cmd.PersistentFlags().MarkDeprecated("metrics-port", "Use metrics-ports instead")

	cmd.AddCommand(newAWSPrepareCommand())
	cmd.AddCommand(newGCPPrepareCommand())
	cmd.AddCommand(newRHOSPrepareCommand())
	cmd.AddCommand(newGenericPrepareCommand())

	return cmd
}

type metricsAliasType struct{}

func (m metricsAliasType) String() string {
	return strconv.FormatUint(uint64(metricsPort[0]), 10)
}

func (m metricsAliasType) Set(s string) error {
	v, err := strconv.ParseUint(s, 0, 16)
	if err != nil {
		return errors.Wrap(err, "conversion to uint16 failed")
	}

	metricsPort = []uint16{uint16(v)}

	return nil
}

func (m metricsAliasType) Type() string {
	return "uint16"
}
