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
	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
)

var parentRestConfigProducer *restconfig.Producer

// NewCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func NewCommand(restConfigProducer *restconfig.Producer) *cobra.Command {
	parentRestConfigProducer = restConfigProducer

	cmd := &cobra.Command{
		Use:   "vpc-peering",
		Short: "Manage VPC Peering between clusters",
		Long:  `This command creates a VPC Peering between different clusters of the same cloud provider.`,
	}

	cmd.AddCommand(newCreateCommand(restConfigProducer))
	cmd.AddCommand(newCleanupCommand(restConfigProducer))

	return cmd
}

func newCreateCommand(restConfigProducer *restconfig.Producer) *cobra.Command {
	parentRestConfigProducer = restConfigProducer

	// Create VPC-Peering
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a VPC Peering between clusters",
		Long:  `This command creates a VPC Peering between different clusters of the same cloud provider.`,
	}

	cmd.AddCommand(newAWSVPCPeeringCommand())
	cmd.AddCommand(newGCPVPCPeeringCommand())
	cmd.AddCommand(newGenericVPCPeeringCommand())

	return cmd
}

func newCleanupCommand(restConfigProducer *restconfig.Producer) *cobra.Command {
	parentRestConfigProducer = restConfigProducer

	// Clean up VPC-Peering
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Remove a VPC Peering between clusters",
		Long:  `This command removes a VPC Peering between different clusters of the same cloud provider.`,
	}

	cmd.AddCommand(newCleanupAWSVPCPeeringCommand())
	cmd.AddCommand(newCleanupGCPVPCPeeringCommand())
	cmd.AddCommand(newCleanupGenericVPCPeeringCommand())

	return cmd
}
