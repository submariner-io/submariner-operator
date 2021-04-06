/*
© 2021 Red Hat, Inc. and others.

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
	cloudutils "github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
)

var (
	gwInstanceType string
	infraID        string
	region         string
	gateways       int
)

// NewCommand returns a new cobra.Command used to prepare a cloud infrastructure
func newAWSPrepareCommand() *cobra.Command {
	awsCloudPrepareCmd := &cobra.Command{
		Use:   "aws",
		Short: "Prepare an AWS cloud",
		Long:  "This command prepares an AWS based cloud for Submariner installation.",
		Run:   prepareAws,
	}

	awsCloudPrepareCmd.Flags().StringVar(&gwInstanceType, "gateway-instance", "m5n.large", "Type of gateways instance machine")
	awsCloudPrepareCmd.Flags().StringVar(&infraID, "infra-id", "", "AWS infra ID")
	awsCloudPrepareCmd.Flags().StringVar(&region, "region", "", "AWS region")
	awsCloudPrepareCmd.Flags().IntVar(&gateways, "gateways", 1, "Amount of gateways to prepare (0 = gateway per public subnet)")

	return awsCloudPrepareCmd
}

func prepareAws(cmd *cobra.Command, args []string) {
	input := api.PrepareForSubmarinerInput{
		InternalPorts: []api.PortSpec{
			{Port: vxlanPort, Protocol: "udp"},
			{Port: metricsPort, Protocol: "tcp"},
		},
		PublicPorts: []api.PortSpec{
			{Port: nattPort, Protocol: "udp"},
			{Port: natDiscoveryPort, Protocol: "udp"},
		},
		Gateways: gateways,
	}
	err := cloudutils.RunOnAWS(infraID, region, gwInstanceType, *kubeConfig, *kubeContext,
		func(cloud api.Cloud, reporter api.Reporter) error {
			return cloud.PrepareForSubmariner(input, reporter)
		})

	utils.ExitOnError("Failed to prepare AWS cloud", err)
}
