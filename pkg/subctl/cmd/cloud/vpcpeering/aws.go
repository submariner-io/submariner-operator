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

package vpcpeering

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	cloudprepareaws "github.com/submariner-io/cloud-prepare/pkg/aws"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/aws"
	cloudutils "github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/utils"
)

var targetArgs = aws.NewArgs("target")

// NewCommand returns a new cobra.Command used to create a VPC Peering on a cloud infrastructure.
func newAWSVPCPeeringCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aws",
		Short: "Create a VPC Peering on AWS cloud",
		Long:  "This command prepares an OpenShift installer-provisioned infrastructure (IPI) on AWS cloud for Submariner installation.",
		Run:   vpcPeerAws,
	}

	aws.ClientArgs.AddAWSFlags(cmd)
	targetArgs.AddAWSFlags(cmd)

	return cmd
}

func vpcPeerAws(cmd *cobra.Command, args []string) {
	targetArgs.ValidateFlags()

	reporter := cloudutils.NewStatusReporter()

	reporter.Started("Initializing AWS connectivity")

	targetCloud, err := cloudprepareaws.NewCloudFromSettings(targetArgs.CredentialsFile,
		targetArgs.Profile, targetArgs.InfraID, targetArgs.Region)
	if err != nil {
		reporter.Failed(err)
		exit.OnErrorWithMessage(err, "Failed to initialize AWS connectivity")
	}

	reporter.Succeeded("")

	err = aws.ClientArgs.RunOnAWS(*parentRestConfigProducer, "",
		func(cloud api.Cloud, gwDeployer api.GatewayDeployer, reporter api.Reporter) error {
			return errors.Wrap(err, cloud.CreateVpcPeering(targetCloud, reporter).Error())
		})
	if err != nil {
		exit.OnErrorWithMessage(err, "Failed to create VPC Peering on AWS cloud")
	}
}

// newCleanupAWSVPCPeeringCommand removes a VPC Peering between different AWS clusters.
func newCleanupAWSVPCPeeringCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aws",
		Short: "Removes VPC Peering on AWS cloud",
		Long:  "This command cleans an OpenShift installer-provisioned infrastructure (IPI) on AWS cloud for Submariner uninstallation.",
		Run:   cleanupVpcPeerAws,
	}

	aws.ClientArgs.AddAWSFlags(cmd)
	targetArgs.AddAWSFlags(cmd)

	return cmd
}

// cleanupVpcPeerAws removes peering object and routes between two OCP clusters in AWS.
func cleanupVpcPeerAws(cmd *cobra.Command, args []string) {
	targetArgs.ValidateFlags()

	reporter := cloudutils.NewStatusReporter()

	reporter.Started("Initializing AWS connectivity")
	reporter.Succeeded("")

	var err error

	err = aws.ClientArgs.RunOnAWS(*parentRestConfigProducer, "",
		func(cloud api.Cloud, gwDeployer api.GatewayDeployer, reporter api.Reporter) error {
			return errors.Wrap(err, cloud.CleanupAfterSubmariner(reporter).Error())
		})
	if err != nil {
		exit.OnErrorWithMessage(err, "Failed to remove VPC Peering on AWS cloud")
	}
}
