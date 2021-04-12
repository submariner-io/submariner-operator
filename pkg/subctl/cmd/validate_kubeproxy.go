/*
© 2021 Red Hat, Inc. and others.

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
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	"github.com/submariner-io/shipyard/test/e2e/framework"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/resource"
)

const (
	KubeProxyIPVSIfaceCommand = "ip a s kube-ipvs0"
	MissingInterface          = "ip: can't find device"
)

var (
	namespace string
)

var validateKubeProxyModeCmd = &cobra.Command{
	Use:   "kube-proxy-mode",
	Short: "Validate the kube-proxy mode.",
	Long:  "This command checks if the kube-proxy mode is supported by Submariner.",
	Run:   validateKubeProxyMode,
}

func init() {
	validateKubeProxyModeCmd.Flags().StringVar(&namespace, "namespace", "default",
		"namespace in which validation pods should be deployed")
	validateCmd.AddCommand(validateKubeProxyModeCmd)
}

func validateKubeProxyMode(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContexts)
	exitOnError("Error getting REST config for cluster", err)

	for _, item := range configs {
		message := fmt.Sprintf("Validating Submariner support for the kube-proxy mode"+
			" used in cluster %q", item.clusterName)
		status.Start(message)

		clientset, err := kubernetes.NewForConfig(item.config)
		if err != nil {
			message := fmt.Sprintf("Error creating API server client: %s", err)
			status.QueueFailureMessage(message)
			status.End(cli.Failure)
			return
		}

		podOutput, err := resource.SchedulePodAwaitCompletion(&resource.PodConfig{
			Name:       "query-iface-list",
			ClientSet:  clientset,
			Scheduling: framework.GatewayNode,
			Networking: framework.HostNetworking,
			Namespace:  namespace,
			Command:    KubeProxyIPVSIfaceCommand,
		})

		if err != nil {
			message := fmt.Sprintf("Error while spawning the Network Pod. %v", err)
			status.QueueFailureMessage(message)
			status.End(cli.Failure)
			return
		}

		if strings.Contains(podOutput, MissingInterface) {
			status.QueueSuccessMessage("Cluster is not deployed with kube-proxy ipvs mode.")
			status.End(cli.Success)
		} else {
			status.QueueFailureMessage("Cluster is deployed with kube-proxy ipvs mode." +
				" Submariner does not support this mode.")
			status.End(cli.Failure)
		}
	}
}
