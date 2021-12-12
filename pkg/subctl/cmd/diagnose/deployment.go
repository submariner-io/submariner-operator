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
	"github.com/submariner-io/submariner/pkg/cidr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func init() {
	diagnoseCmd.AddCommand(&cobra.Command{
		Use:   "deployment",
		Short: "Check the Submariner deployment",
		Long:  "This command checks that the Submariner components are properly deployed and running with no overlapping CIDRs.",
		Run: func(command *cobra.Command, args []string) {
			cmd.ExecuteMultiCluster(func(cluster *cmd.Cluster) bool {
				if cluster.Submariner == nil {
					status := cli.NewStatus()
					status.Start(cmd.SubmMissingMessage)
					status.End(cli.Warning)
					return true
				}

				return checkOverlappingCIDRs(cluster) && checkPods(cluster)
			})
		},
	})
}

func checkOverlappingCIDRs(cluster *cmd.Cluster) bool {
	status := cli.NewStatus()

	if cluster.Submariner.Spec.GlobalCIDR != "" {
		status.Start("Globalnet deployment detected - checking if globalnet CIDRs overlap")
	} else {
		status.Start("Non-Globalnet deployment detected - checking if cluster CIDRs overlap")
	}

	endpointList, err := cluster.SubmClient.SubmarinerV1().Endpoints(cluster.Submariner.Namespace).List(context.TODO(),
		metav1.ListOptions{})
	if err != nil {
		status.EndWithFailure("Error listing the Submariner endpoints: %v", err)
		return false
	}

	for i := range endpointList.Items {
		source := &endpointList.Items[i]

		destEndpoints := endpointList.Items[i+1:]
		for j := range destEndpoints {
			dest := &destEndpoints[j]

			// Currently we dont support multiple endpoints in a cluster, hence return an error.
			// When the corresponding support is added, this check needs to be updated.
			if source.Spec.ClusterID == dest.Spec.ClusterID {
				status.QueueFailureMessage(fmt.Sprintf("Found multiple Submariner endpoints (%q and %q) in cluster %q",
					source.Name, dest.Name, source.Spec.ClusterID))
				continue
			}

			for _, subnet := range dest.Spec.Subnets {
				overlap, err := cidr.IsOverlapping(source.Spec.Subnets, subnet)
				if err != nil {
					// Ideally this case will never hit, as the subnets are valid CIDRs
					status.QueueFailureMessage(fmt.Sprintf("Error parsing CIDR in cluster %q: %s", dest.Spec.ClusterID, err))
					continue
				}

				if overlap {
					status.QueueFailureMessage(fmt.Sprintf("CIDR %q in cluster %q overlaps with cluster %q (CIDRs: %v)",
						subnet, dest.Spec.ClusterID, source.Spec.ClusterID, source.Spec.Subnets))
				}
			}
		}
	}

	if status.HasFailureMessages() {
		status.End(cli.Failure)
		return false
	}

	if cluster.Submariner.Spec.GlobalCIDR != "" {
		status.EndWithSuccess("Clusters do not have overlapping globalnet CIDRs")
	} else {
		status.EndWithSuccess("Clusters do not have overlapping CIDRs")
	}

	return true
}

func checkPods(cluster *cmd.Cluster) bool {
	status := cli.NewStatus()
	status.Start("Checking Submariner pods")

	checkDaemonset(cluster.KubeClient, cmd.OperatorNamespace, "submariner-gateway", status)
	checkDaemonset(cluster.KubeClient, cmd.OperatorNamespace, "submariner-routeagent", status)

	// Check if service-discovery components are deployed and running if enabled
	if cluster.Submariner.Spec.ServiceDiscoveryEnabled {
		checkDeployment(cluster.KubeClient, cmd.OperatorNamespace, "submariner-lighthouse-agent", status)
		checkDeployment(cluster.KubeClient, cmd.OperatorNamespace, "submariner-lighthouse-coredns", status)
	}

	// Check if globalnet components are deployed and running if enabled
	if cluster.Submariner.Spec.GlobalCIDR != "" {
		checkDaemonset(cluster.KubeClient, cmd.OperatorNamespace, "submariner-globalnet", status)
	}

	checkPodsStatus(cluster.KubeClient, cmd.OperatorNamespace, status)

	if status.HasFailureMessages() {
		status.End(cli.Failure)
		return false
	}

	status.EndWithSuccess("All Submariner pods are up and running")

	return true
}

func checkDeployment(k8sClient kubernetes.Interface, namespace, deploymentName string, status *cli.Status) {
	deployment, err := k8sClient.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		status.QueueFailureMessage(fmt.Sprintf("Error obtaining Deployment %q: %v", deploymentName, err))
		return
	}

	var replicas int32 = 1
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	if deployment.Status.AvailableReplicas != replicas {
		status.QueueFailureMessage(fmt.Sprintf("The desired number of replicas for Deployment %q (%d)"+
			" does not match the actual number running (%d)", deploymentName, replicas,
			deployment.Status.AvailableReplicas))
	}
}

func checkDaemonset(k8sClient kubernetes.Interface, namespace, daemonSetName string, status *cli.Status) {
	daemonSet, err := k8sClient.AppsV1().DaemonSets(namespace).Get(context.TODO(), daemonSetName, metav1.GetOptions{})
	if err != nil {
		status.QueueFailureMessage(fmt.Sprintf("Error obtaining Daemonset %q: %v", daemonSetName, err))
		return
	}

	if daemonSet.Status.CurrentNumberScheduled != daemonSet.Status.DesiredNumberScheduled {
		status.QueueFailureMessage(fmt.Sprintf("The desired number of running pods for DaemonSet %q (%d)"+
			" does not match the actual number (%d)", daemonSetName, daemonSet.Status.DesiredNumberScheduled,
			daemonSet.Status.CurrentNumberScheduled))
	}
}

func checkPodsStatus(k8sClient kubernetes.Interface, namespace string, status *cli.Status) {
	pods, err := k8sClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		status.QueueFailureMessage(fmt.Sprintf("Error obtaining Pods list: %v", err))
		return
	}

	for i := range pods.Items {
		pod := &pods.Items[i]
		if pod.Status.Phase != v1.PodRunning {
			status.QueueFailureMessage(fmt.Sprintf("Pod %q is not running. (current state is %v)", pod.Name, pod.Status.Phase))
			continue
		}

		for j := range pod.Status.ContainerStatuses {
			c := &pod.Status.ContainerStatuses[j]
			if c.RestartCount >= 5 {
				status.QueueWarningMessage(fmt.Sprintf("Pod %q has restarted %d times", pod.Name, c.RestartCount))
			}
		}
	}
}
