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
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/internal/pods"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
	subv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	ValidationTimeout uint
	VerboseOutput     bool
)

func spawnSnifferPodOnGatewayNode(client kubernetes.Interface, namespace, podCommand string) (*pods.Scheduled, error) {
	scheduling := pods.Scheduling{ScheduleOn: pods.GatewayNode, Networking: pods.HostNetworking}
	return spawnPod(client, scheduling, "validate-sniffer", namespace, podCommand)
}

func spawnSnifferPodOnNode(client kubernetes.Interface, nodeName, namespace, podCommand string) (*pods.Scheduled, error) {
	scheduling := pods.Scheduling{
		ScheduleOn: pods.CustomNode, NodeName: nodeName,
		Networking: pods.HostNetworking,
	}

	return spawnPod(client, scheduling, "validate-sniffer", namespace, podCommand)
}

func spawnClientPodOnNonGatewayNode(client kubernetes.Interface, namespace, podCommand string) (*pods.Scheduled, error) {
	scheduling := pods.Scheduling{ScheduleOn: pods.NonGatewayNode, Networking: pods.PodNetworking}
	return spawnPod(client, scheduling, "validate-client", namespace, podCommand)
}

func spawnPod(client kubernetes.Interface, scheduling pods.Scheduling, podName, namespace,
	podCommand string) (*pods.Scheduled, error) {
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

func getActiveGatewayNodeName(clusterInfo *cluster.Info, hostname string, status reporter.Interface) (string, bool) {
	nodes, err := clusterInfo.ClientProducer.ForKubernetes().CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: "submariner.io/gateway=true",
	})
	if err != nil {
		status.Failure("Error obtaining the Gateway Nodes in cluster %q: %v", clusterInfo.Name, err)
		return "", true
	}

	for i := range nodes.Items {
		node := &nodes.Items[i]
		if node.Name == hostname {
			return hostname, false
		}

		// On some platforms, the nodeName does not match with the hostname.
		// Submariner Endpoint stores the hostname info in the endpoint and not the nodeName. So, we spawn a
		// tiny pod to read the hostname and return the corresponding node.
		sPod, err := spawnSnifferPodOnNode(clusterInfo.ClientProducer.ForKubernetes(), node.Name, "default", "hostname")
		if err != nil {
			status.Failure("Error spawning the sniffer pod on the node %q: %v", node.Name, err)
			return "", true
		}

		defer sPod.Delete()

		if err = sPod.AwaitCompletion(); err != nil {
			status.Failure("Error waiting for the sniffer pod to finish its execution on node %q: %v", node.Name, err)
			return "", true
		}

		if sPod.PodOutput[:len(sPod.PodOutput)-1] == hostname {
			return node.Name, false
		}
	}

	status.Failure("Could not find the active Gateway node %q in local cluster in cluster %q",
		hostname, clusterInfo.Name)

	return "", true
}

func getLocalEndpointResource(clusterInfo *cluster.Info, status reporter.Interface) (*subv1.Endpoint, bool) {
	endpoints, err := clusterInfo.ClientProducer.ForSubmariner().SubmarinerV1().Endpoints(constants.OperatorNamespace).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		status.Failure("Error obtaining the Endpoints in cluster %q: %v", clusterInfo.Name, err)
		return nil, true // failed = true
	}

	for i := range endpoints.Items {
		if endpoints.Items[i].Spec.ClusterID == clusterInfo.Submariner.Spec.ClusterID {
			return &endpoints.Items[i], false
		}
	}

	status.Failure("Could not find the local Endpoint in cluster %q", clusterInfo.Name)

	return nil, true
}

func getAnyRemoteEndpointResource(clusterInfo *cluster.Info, status reporter.Interface) (*subv1.Endpoint, bool) {
	endpoints, err := clusterInfo.ClientProducer.ForSubmariner().SubmarinerV1().Endpoints(constants.OperatorNamespace).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		status.Failure("Error obtaining the Endpoints in cluster %q: %v", clusterInfo.Name, err)
		return nil, true
	}

	for i := range endpoints.Items {
		if endpoints.Items[i].Spec.ClusterID != clusterInfo.Submariner.Spec.ClusterID {
			return &endpoints.Items[i], false
		}
	}

	status.Failure("Could not find any remote Endpoint in cluster %q", clusterInfo.Name)

	return nil, true
}
