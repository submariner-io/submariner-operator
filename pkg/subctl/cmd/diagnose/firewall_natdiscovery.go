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

package diagnose

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/cli"
)

func init() {
	command := &cobra.Command{
		Use:   "nat-discovery <localkubeconfig> <remotekubeconfig>",
		Short: "Check firewall access for nat-discovery to function properly",
		Long:  "This command checks if the firewall configuration allows nat-discovery between the configured Gateway nodes.",
		Args:  checkKubeconfigArgs,
		Run:   validateNatDiscoveryPort,
	}

	addDiagnoseFWConfigFlags(command)
	addVerboseFlag(command)
	diagnoseFirewallConfigCmd.AddCommand(command)
}

func validateNatDiscoveryPort(command *cobra.Command, args []string) {
	localCluster, remoteCluster := getClusterDetails(args)

	status := cli.NewStatus()
	status.Start(fmt.Sprintf("Checking if nat-discovery port is opened on the gateway node of cluster %q", localCluster.Name))

	if isClusterSingleNode(remoteCluster, status) {
		// Skip the check if it's a single node cluster
		return
	}

	if !validateConnectivity(localCluster, remoteCluster, NatDiscoveryPort, status) {
		status.EndWithFailure("Could not determine if nat-discovery port is allowed in the cluster %q", localCluster.Name)
		os.Exit(1)
	}

	status.EndWithSuccess("nat-discovery port is allowed in the cluster %q", localCluster.Name)
}
