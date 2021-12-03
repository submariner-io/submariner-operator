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
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	diagnoseCmd.AddCommand(&cobra.Command{
		Use:   "all",
		Short: "Run all diagnostic checks (except those requiring two kubecontexts)",
		Long:  "This command runs all diagnostic checks (except those requiring two kubecontexts) and reports any issues",
		Run: func(command *cobra.Command, args []string) {
			cmd.ExecuteMultiCluster(diagnoseAll)
		},
	})
}

func diagnoseAll(cluster *cmd.Cluster) bool {
	success := checkK8sVersion(cluster)
	fmt.Println()

	status := cli.NewStatus()
	if cluster.Submariner == nil {
		status.Start(cmd.SubmMissingMessage)
		status.End(cli.Warning)
		return success
	}

	success = checkCNIConfig(cluster) && success
	fmt.Println()

	success = checkConnections(cluster) && success
	fmt.Println()

	success = checkPods(cluster) && success
	fmt.Println()

	success = checkOverlappingCIDRs(cluster) && success
	fmt.Println()

	success = checkKubeProxyMode(cluster) && success
	fmt.Println()

	success = checkFirewallMetricsConfig(cluster) && success
	fmt.Println()

	success = checkVxLANConfig(cluster) && success
	fmt.Println()

	fmt.Printf("Skipping inter-cluster firewall check as it requires two kubeconfigs." +
		" Please run \"subctl diagnose firewall inter-cluster\" command manually.\n")

	return success
}

func getNumNodesOfCluster(cluster *cmd.Cluster) (int, error) {
	nodes, err := cluster.KubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}

	return len(nodes.Items), nil
}

func isClusterSingleNode(cluster *cmd.Cluster, status *cli.Status) bool {
	numNodesOfCluster, err := getNumNodesOfCluster(cluster)
	if err != nil {
		status.EndWithFailure("Error listing the number of nodes of the cluster: %v", err)
		return true
	}

	if numNodesOfCluster == 1 {
		status.EndWithSuccess("Skipping this check as it's a single node cluster.")
		return true
	}

	return false
}
