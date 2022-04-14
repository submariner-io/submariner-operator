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

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/pods"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
)

const (
	kubeProxyIPVSIfaceCommand = "ip a s kube-ipvs0"
	missingInterface          = "ip: can't find device"
)

func init() {
	command := &cobra.Command{
		Use:   "kube-proxy-mode",
		Short: "Check the kube-proxy mode",
		Long:  "This command checks if the kube-proxy mode is supported by Submariner.",
		Run: func(command *cobra.Command, args []string) {
			cmd.ExecuteMultiCluster(restConfigProducer, checkKubeProxyMode)
		},
	}

	addNamespaceFlag(command)
	diagnoseCmd.AddCommand(command)
}

func checkKubeProxyMode(cluster *cmd.Cluster) bool {
	status := cli.NewStatus()
	status.Start("Checking Submariner support for the kube-proxy mode")

	scheduling := pods.Scheduling{ScheduleOn: pods.GatewayNode, Networking: pods.HostNetworking}

	podOutput, err := pods.ScheduleAndAwaitCompletion(&pods.Config{
		Name:       "query-iface-list",
		ClientSet:  cluster.KubeClient,
		Scheduling: scheduling,
		Namespace:  podNamespace,
		Command:    kubeProxyIPVSIfaceCommand,
	})
	if err != nil {
		status.EndWithFailure("Error spawning the network pod: %v", err)
		return false
	}

	if strings.Contains(podOutput, missingInterface) {
		status.QueueSuccessMessage("The kube-proxy mode is supported")
	} else {
		status.QueueFailureMessage("The cluster is deployed with kube-proxy ipvs mode which Submariner does not support")
	}

	result := status.ResultFromMessages()
	status.EndWith(result)

	return result != cli.Failure
}
