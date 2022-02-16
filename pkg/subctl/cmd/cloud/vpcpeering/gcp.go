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
	cloudpreparegcp "github.com/submariner-io/cloud-prepare/pkg/gcp"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/gcp"
	cloudutils "github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/utils"
)

// NewCommand returns a new cobra.Command used to create a VPC Peering on a cloud infrastructure.
func newGCPVPCPeeringCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gcp",
		Short: "Create a VPC Peering on GCP cloud",
		Long:  "This command prepares an OpenShift installer-provisioned infrastructure (IPI) on GCP cloud for Submariner installation.",
		Run:   vpcPeerGcp,
	}

	gcp.ClientArgs.AddGCPFlags(cmd)
	targetArgs.AddAWSFlags(cmd)

	return cmd
}

func vpcPeerGcp(cmd *cobra.Command, args []string) {
	targetArgs.ValidateFlags()

	reporter := cloudutils.NewStatusReporter()

	reporter.Started("Initializing GCP connectivity")

	cloudInfo, err := cloudpreparegcp.NewCloudInfoFromSettings(targetArgs.CredentialsFile,
		targetArgs.Profile, targetArgs.InfraID, targetArgs.Region)
	if err != nil {
		reporter.Failed(err)
		exit.OnErrorWithMessage(err, "Failed to initialize GCP connectivity")
	}

	reporter.Succeeded("")

	err = gcp.ClientArgs.RunOnGCP(*parentRestConfigProducer, "", false,
		func(cloud api.Cloud, gwDeployer api.GatewayDeployer, reporter api.Reporter) error {
			return errors.Wrap(err, cloud.CreateVpcPeering(cloudpreparegcp.NewCloud(*cloudInfo), reporter).Error())
		})
	if err != nil {
		exit.OnErrorWithMessage(err, "Failed to create VPC Peering on GCP cloud")
	}
}

// newCleanupAWSVPCPeeringCommand removes a VPC Peering between different AWS clusters.
func newCleanupGCPVPCPeeringCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gcp",
		Short: "Removes VPC Peering on GCP cloud",
		Long:  "This command cleans an OpenShift installer-provisioned infrastructure (IPI) on GCP cloud for Submariner uninstallation.",
		Run:   cleanupVpcPeerGcp,
	}

	gcp.ClientArgs.AddGCPFlags(cmd)
	targetArgs.AddAWSFlags(cmd)

	return cmd
}

// cleanupVpcPeerGcp removes peering object and routes between two OCP clusters in AWS.
func cleanupVpcPeerGcp(cmd *cobra.Command, args []string) {
	targetArgs.ValidateFlags()

	reporter := cloudutils.NewStatusReporter()

	reporter.Started("Initializing GCP connectivity")

	reporter.Succeeded("")

	var err error

	err = gcp.ClientArgs.RunOnGCP(*parentRestConfigProducer, "", false,
		func(cloud api.Cloud, gwDeployer api.GatewayDeployer, reporter api.Reporter) error {
			return errors.Wrap(err, cloud.CleanupAfterSubmariner(reporter).Error())
		})
	if err != nil {
		exit.OnErrorWithMessage(err, "Failed to create VPC Peering on GCP cloud")
	}
}
