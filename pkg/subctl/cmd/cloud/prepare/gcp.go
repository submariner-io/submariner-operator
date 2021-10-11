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
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/gcp"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
)

// NewCommand returns a new cobra.Command used to prepare a cloud infrastructure
func newGCPPrepareCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gcp",
		Short: "Prepare an OpenShift GCP cloud",
		Long:  "This command prepares an OpenShift installer-provisioned infrastructure (IPI) on GCP cloud for Submariner installation.",
		Run:   prepareGCP,
	}

	gcp.AddGCPFlags(cmd)
	cmd.Flags().StringVar(&gcpGWInstanceType, "gateway-instance", "n1-standard-4", "Type of gateway instance machine")
	cmd.Flags().IntVar(&gateways, "gateways", DefaultNumGateways,
		"Number of gateways to deploy")
	cmd.Flags().BoolVar(&dedicatedGateway, "dedicated-gateway", false,
		"Whether a dedicated gateway node has to be deployed (default false)")
	return cmd
}

func prepareGCP(cmd *cobra.Command, args []string) {
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

	err := gcp.RunOnGCP(gcpGWInstanceType, *kubeConfig, *kubeContext, dedicatedGateway,
		func(cloud api.Cloud, gwDeployer api.GatewayDeployer, reporter api.Reporter) error {
			if gateways > 0 {
				gwInput := api.GatewayDeployInput{
					PublicPorts: gwPorts,
					Gateways:    gateways,
				}
				err := gwDeployer.Deploy(gwInput, reporter)
				if err != nil {
					return err
				}
			}

			return cloud.PrepareForSubmariner(input, reporter)
		})

	utils.ExitOnError("Failed to prepare GCP cloud", err)
}
