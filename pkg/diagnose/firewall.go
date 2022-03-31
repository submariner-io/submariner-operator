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
	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/internal/pods"
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
