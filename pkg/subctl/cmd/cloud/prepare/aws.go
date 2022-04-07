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
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/aws"
)

// NewCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func newAWSPrepareCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aws",
		Short: "Prepare an OpenShift AWS cloud",
		Long:  "This command prepares an OpenShift installer-provisioned infrastructure (IPI) on AWS cloud for Submariner installation.",
		Run:   prepareAws,
	}

	aws.AddAWSFlags(cmd)
	cmd.Flags().StringVar(&awsGWInstanceType, "gateway-instance", "c5d.large", "Type of gateways instance machine")
	cmd.Flags().IntVar(&gateways, "gateways", DefaultNumGateways,
		"Number of dedicated gateways to deploy (Set to `0` when using --load-balancer mode)")

	return cmd
}

func prepareAws(cmd *cobra.Command, args []string) {
	gwPorts := []api.PortSpec{
		{Port: nattPort, Protocol: "udp"},
		{Port: natDiscoveryPort, Protocol: "udp"},

		// ESP & AH protocols are used for private-ip to private-ip gateway communications.
		{Port: 0, Protocol: "50"},
		{Port: 0, Protocol: "51"},
	}
	input := api.PrepareForSubmarinerInput{
		InternalPorts: []api.PortSpec{
			{Port: vxlanPort, Protocol: "udp"},
			{Port: metricsPort, Protocol: "tcp"},
		},
	}

	// For load-balanced gateways we want these ports open internally to facilitate private-ip to pivate-ip gateways communications.
	if gateways == 0 {
		input.InternalPorts = append(input.InternalPorts, gwPorts...)
	}

	// nolint:wrapcheck // No need to wrap errors here.
	err := aws.RunOnAWS(*parentRestConfigProducer, awsGWInstanceType, cli.NewReporter(),
		func(cloud api.Cloud, gwDeployer api.GatewayDeployer, status reporter.Interface) error {
			if gateways > 0 {
				gwInput := api.GatewayDeployInput{
					PublicPorts: gwPorts,
					Gateways:    gateways,
				}
				err := gwDeployer.Deploy(gwInput, status)
				if err != nil {
					return err
				}
			}

			return cloud.PrepareForSubmariner(input, status)
		})

	exit.OnErrorWithMessage(err, "Failed to prepare AWS cloud")
}
