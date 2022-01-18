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

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func All(clusterInfo *cluster.Info, status reporter.Interface) bool {
	success := K8sVersion(clusterInfo, status)

	fmt.Println()

	if clusterInfo.Submariner == nil {
		status.Warning(constants.SubmMissingMessage)

		return success
	}

	success = CNIConfig(clusterInfo, status) && success

	fmt.Println()

	success = Connections(clusterInfo, status) && success

	fmt.Println()

	success = Deployments(clusterInfo, status) && success

	fmt.Println()

	success = KubeProxyMode(clusterInfo, status) && success

	fmt.Println()

	success = FirewallMetricsConfig(clusterInfo, status) && success

	fmt.Println()

	success = VxLANConfig(clusterInfo, status) && success

	fmt.Println()

	fmt.Printf("Skipping inter-cluster firewall check as it requires two kubeconfigs." +
		" Please run \"subctl diagnose firewall inter-cluster\" command manually.\n")

	return success
}

func getNumNodesOfCluster(clusterInfo *cluster.Info) (int, error) {
	nodes, err := clusterInfo.ClientProducer.ForKubernetes().CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, errors.Wrap(err, "error listing Nodes")
	}

	return len(nodes.Items), nil
}

func isClusterSingleNode(clusterInfo *cluster.Info, status reporter.Interface) bool {
	numNodesOfCluster, err := getNumNodesOfCluster(clusterInfo)
	if err != nil {
		status.Failure("Error listing the number of nodes of the cluster: %v", err)
		return true
	}

	if numNodesOfCluster == 1 {
		status.Success("Skipping this check as it's a single node cluster.")
		return true
	}

	return false
}
