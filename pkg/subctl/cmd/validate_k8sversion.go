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
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
)

var validateK8sVersionCmd = &cobra.Command{
	Use:   "k8s-version",
	Short: "Check the Kubernetes version",
	Long:  "This command checks if Submariner can be deployed on the Kubernetes version.",
	Run:   validateK8sVersion,
}

func init() {
	validateCmd.AddCommand(validateK8sVersionCmd)
}

func validateK8sVersion(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContexts)
	exitOnError("Error getting REST config for cluster", err)

	validationStatus := true

	for _, item := range configs {
		validationStatus = validationStatus && validateK8sVersionInCluster(item.config, item.clusterName)
	}
	if !validationStatus {
		os.Exit(1)
	}
}

func validateK8sVersionInCluster(config *rest.Config, clusterName string) bool {
	message := fmt.Sprintf("Checking Submariner support for the Kubernetes version"+
		" used in cluster %q", clusterName)
	status.Start(message)

	failedRequirements, err := checkRequirements(config)
	if len(failedRequirements) > 0 {
		status.QueueFailureMessage("The Kubernetes version does not meet Submariner's requirements:")
		for i := range failedRequirements {
			message = fmt.Sprintf("* %s\n", (failedRequirements)[i])
			status.QueueFailureMessage(message)
		}
		status.End(cli.Failure)
		return false
	}
	if err != nil {
		status.QueueFailureMessage(err.Error())
		status.End(cli.Failure)
		return false
	}
	status.QueueSuccessMessage("The Kubernetes version meets Submariner's requirements")
	status.End(cli.Success)
	return true
}
