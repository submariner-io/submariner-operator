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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/azure"
)

// newAzurePrepareCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func newAzurePrepareCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "azure",
		Short: "Prepare an OpenShift Azure cloud",
		Long:  "This command prepares an OpenShift installer-provisioned infrastructure (IPI) on Azure cloud for Submariner installation.",
		Run:   prepareAzure,
	}

	azure.AddAzureFlags(cmd)
	cmd.Flags().IntVar(&gateways, "gateways", DefaultNumGateways,
		"Number of gateways to deploy")
	// `Standard_F4s_v2` matches the most to `cd5.large` of AWS.
	cmd.Flags().StringVar(&azureGWInstanceType, "gateway-instance", "Standard_F4s_v2", "Type of gateways instance machine")

	return cmd
}

func prepareAzure(cmd *cobra.Command, args []string) {
	gwPorts := []api.PortSpec{
		{Port: nattPort, Protocol: "udp"},
		{Port: natDiscoveryPort, Protocol: "udp"},

		// ESP & AH protocols are used for private-ip to private-ip gateway communications
		{Port: 0, Protocol: "esp"},
		{Port: 0, Protocol: "ah"},
	}

	input := api.PrepareForSubmarinerInput{
		InternalPorts: []api.PortSpec{
			{Port: vxlanPort, Protocol: "udp"},
			{Port: metricsPort, Protocol: "tcp"},
		},
	}

	// nolint:wrapcheck // No need to wrap errors here.
	err := azure.RunOnAzure(*parentRestConfigProducer, azureGWInstanceType, cli.NewReporter(),
		func(cloud api.Cloud, gwDeployer api.GatewayDeployer, status reporter.Interface) error {
			if gateways > 0 {
				gwInput := api.GatewayDeployInput{
					PublicPorts: gwPorts,
					Gateways:    gateways,
				}

				err := gwDeployer.Deploy(gwInput, status)
				if err != nil {
					return errors.WithMessage(err, "Deployment failed ")
				}
			}

			return cloud.PrepareForSubmariner(input, status)
		})

	exit.OnErrorWithMessage(err, "Failed to prepare Azure  cloud")
}
