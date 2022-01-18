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
	"strings"

	"github.com/submariner-io/submariner-operator/internal/pods"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
)

var KubeProxyPodNamespace string

const (
	kubeProxyIPVSIfaceCommand = "ip a s kube-ipvs0"
	missingInterface          = "ip: can't find device"
)

func KubeProxyMode(clusterInfo *cluster.Info, status reporter.Interface) bool {
	status.Start("Checking Submariner support for the kube-proxy mode")
	defer status.End()

	scheduling := pods.Scheduling{ScheduleOn: pods.GatewayNode, Networking: pods.HostNetworking}

	podOutput, err := pods.ScheduleAndAwaitCompletion(&pods.Config{
		Name:       "query-iface-list",
		ClientSet:  clusterInfo.ClientProducer.ForKubernetes(),
		Scheduling: scheduling,
		Namespace:  KubeProxyPodNamespace,
		Command:    kubeProxyIPVSIfaceCommand,
	})
	if err != nil {
		status.Failure("Error spawning the network pod: %v", err)

		return false
	}

	failed := false

	if strings.Contains(podOutput, missingInterface) {
		status.Success("The kube-proxy mode is supported")
	} else {
		status.Failure("The cluster is deployed with kube-proxy ipvs mode which Submariner does not support")
		failed = true
	}

	return !failed
}
