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

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/pods"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	singleNodeMessage = "Skipping this check as it's a single node cluster"
)

type FirewallOptions struct {
	ValidationTimeout uint
	VerboseOutput     bool
	PodNamespace      string
}

func spawnSnifferPodOnGatewayNode(client kubernetes.Interface, namespace, podCommand string) (*pods.Scheduled, error) {
	scheduling := pods.Scheduling{ScheduleOn: pods.GatewayNode, Networking: pods.HostNetworking}
	return spawnPod(client, scheduling, "validate-sniffer", namespace, podCommand)
}

func spawnClientPodOnNonGatewayNode(client kubernetes.Interface, namespace, podCommand string) (*pods.Scheduled, error) {
	scheduling := pods.Scheduling{ScheduleOn: pods.NonGatewayNode, Networking: pods.PodNetworking}
	return spawnPod(client, scheduling, "validate-client", namespace, podCommand)
}

func spawnPod(client kubernetes.Interface, scheduling pods.Scheduling, podName, namespace,
	podCommand string,
) (*pods.Scheduled, error) {
	pod, err := pods.Schedule(&pods.Config{
		Name:       podName,
		ClientSet:  client,
		Scheduling: scheduling,
		Namespace:  namespace,
		Command:    podCommand,
	})
	if err != nil {
		return nil, errors.Wrap(err, "error scheduling pod")
	}

	return pod, nil
}

func spawnSnifferPodOnNode(client kubernetes.Interface, nodeName, namespace, podCommand string) (*pods.Scheduled, error) {
	scheduling := pods.Scheduling{
		ScheduleOn: pods.CustomNode, NodeName: nodeName,
		Networking: pods.HostNetworking,
	}

	return spawnPod(client, scheduling, "validate-sniffer", namespace, podCommand)
}

func getActiveGatewayNodeName(cluster *cmd.Cluster, hostname string, status *cli.Status) string {
	nodes, err := cluster.KubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: "submariner.io/gateway=true",
	})
	if err != nil {
		status.EndWithFailure("Error obtaining the Gateway Nodes in cluster %q: %v", cluster.Name, err)
		return ""
	}

	for i := range nodes.Items {
		node := &nodes.Items[i]
		if node.Name == hostname {
			return hostname
		}

		// On some platforms, the nodeName does not match with the hostname.
		// Submariner Endpoint stores the hostname info in the endpoint and not the nodeName. So, we spawn a
		// tiny pod to read the hostname and return the corresponding node.
		sPod, err := spawnSnifferPodOnNode(cluster.KubeClient, node.Name, "default", "hostname")
		if err != nil {
			status.EndWithFailure("Error spawning the sniffer pod on the node %q: %v", node.Name, err)
			return ""
		}

		err = sPod.AwaitCompletion()

		sPod.Delete()

		if err != nil {
			status.EndWithFailure("Error waiting for the sniffer pod to finish its execution on node %q: %v", node.Name, err)
			return ""
		}

		if sPod.PodOutput[:len(sPod.PodOutput)-1] == hostname {
			return node.Name
		}
	}

	status.EndWithFailure("Could not find the active Gateway node %q in local cluster in cluster %q",
		hostname, cluster.Name)

	return ""
}
