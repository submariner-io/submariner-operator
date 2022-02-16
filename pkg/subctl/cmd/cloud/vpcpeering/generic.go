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

import "github.com/spf13/cobra"

func newGenericVPCPeeringCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generic",
		Short: "Create a VPC peering between generic clusters for Submariner",
		Long:  "This command labels the required number of gateway nodes for Submariner VPC Peering.",
		Run:   vpcPeerGenericCluster,
	}

	return cmd
}

func vpcPeerGenericCluster(cmd *cobra.Command, args []string) {
	// Nothing to peer here
}

func newCleanupGenericVPCPeeringCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generic",
		Short: "Removes VPC peering between generic clusters for Submariner",
		Long:  "This command create a vpc peering between two OCP cloud agnostic clusters",
		Run:   cleanupVpcPeerGenericCluster,
	}

	return cmd
}

func cleanupVpcPeerGenericCluster(cmd *cobra.Command, args []string) {
	// Nothing to clean up here
}
