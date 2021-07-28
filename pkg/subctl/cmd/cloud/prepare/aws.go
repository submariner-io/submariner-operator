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
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/aws"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
)

var (
	gwInstanceType           string
	gateways                 int
	disableDedicatedGateways bool
)

const DefaultDedicatedGateways = 1

// NewCommand returns a new cobra.Command used to prepare a cloud infrastructure
func newAWSPrepareCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "aws",
		Short:  "Prepare an AWS cloud",
		Long:   "This command prepares an AWS-based cloud for Submariner installation.",
		Run:    prepareAws,
		PreRun: validateArgsAws,
	}

	aws.AddAWSFlags(cmd)
	cmd.Flags().StringVar(&gwInstanceType, "gateway-instance", "m5n.large", "Type of gateways instance machine")
	cmd.Flags().IntVar(&gateways, "gateways", DefaultDedicatedGateways, "Number of gateways to prepare (0 = gateway per public subnet)")
	cmd.Flags().BoolVarP(&disableDedicatedGateways, "disable-gateways", "d", false, "Set up no dedicated gateways "+
		"for use with the --load-balancer mode")
	return cmd
}

func validateArgsAws(cmd *cobra.Command, args []string) {
	if gateways != DefaultDedicatedGateways && disableDedicatedGateways {
		utils.ExitWithErrorMsg("gateways and disable-gateways parameters can't be used together")
	}
}

func prepareAws(cmd *cobra.Command, args []string) {
	input := api.PrepareForSubmarinerInput{
		InternalPorts: []api.PortSpec{
			{Port: vxlanPort, Protocol: "udp"},
			{Port: metricsPort, Protocol: "tcp"},
		},
	}

	gwInput := api.GatewayDeployInput{
		PublicPorts: []api.PortSpec{
			{Port: nattPort, Protocol: "udp"},
			{Port: natDiscoveryPort, Protocol: "udp"},
		},
		Gateways: gateways,
	}
	err := aws.RunOnAWS(gwInstanceType, *kubeConfig, *kubeContext,
		func(cloud api.Cloud, gwDeployer api.GatewayDeployer, reporter api.Reporter) error {
			if !disableDedicatedGateways {
				err := gwDeployer.Deploy(gwInput, reporter)
				if err != nil {
					return err
				}
			}

			return cloud.PrepareForSubmariner(input, reporter)
		})

	utils.ExitOnError("Failed to prepare AWS cloud", err)
}
