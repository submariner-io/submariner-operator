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
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/cloud"
	cloudaws "github.com/submariner-io/submariner-operator/pkg/cloud/aws"
	"github.com/submariner-io/submariner-operator/pkg/cloud/prepare"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/aws"
)

const (
	infraIDFlag = "infra-id"
	regionFlag  = "region"
)

// NewCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func newAWSPrepareCommand(restConfigProducer *restconfig.Producer, ports *cloud.Ports) *cobra.Command {
	awsConfig := &cloudaws.Config{}

	cmd := &cobra.Command{
		Use:   "aws",
		Short: "Prepare an OpenShift AWS cloud",
		Long:  "This command prepares an OpenShift installer-provisioned infrastructure (IPI) on AWS cloud for Submariner installation.",
		Run: func(cmd *cobra.Command, args []string) {
			status := cli.NewReporter()

			var err error
			if awsConfig.OcpMetadataFile != "" {
				awsConfig.InfraID, awsConfig.Region, err = cloudaws.ReadFromFile(awsConfig.OcpMetadataFile)
				exit.OnErrorWithMessage(err, "Failed to read AWS information from OCP metadata file")
			} else {
				expectFlag(infraIDFlag, awsConfig.InfraID)
				expectFlag(regionFlag, awsConfig.Region)
			}

			err = prepare.AWS(restConfigProducer, ports, awsConfig, status)
			exit.OnError(err)
		},
	}

	aws.AddAWSFlags(cmd, awsConfig)
	cmd.Flags().StringVar(&awsConfig.GWInstanceType, "gateway-instance", "c5d.large", "Type of gateways instance machine")
	cmd.Flags().IntVar(&awsConfig.Gateways, "gateways", DefaultNumGateways,
		"Number of dedicated gateways to deploy (Set to `0` when using --load-balancer mode)")

	return cmd
}
