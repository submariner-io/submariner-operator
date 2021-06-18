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
	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/version"

	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
)

func init() {
	diagnoseCmd.AddCommand(&cobra.Command{
		Use:   "k8s-version",
		Short: "Check the Kubernetes version",
		Long:  "This command checks if Submariner can be deployed on the Kubernetes version.",
		Run: func(command *cobra.Command, args []string) {
			cmd.ExecuteMultiCluster(checkK8sVersion)
		},
	})
}

func checkK8sVersion(cluster *cmd.Cluster) bool {
	status := cli.NewStatus()

	status.Start("Checking Submariner support for the Kubernetes version")

	k8sVersion, failedRequirements, err := version.CheckRequirements(cluster.Config)
	if err != nil {
		status.EndWithFailure(err.Error())
		return false
	}

	for i := range failedRequirements {
		status.QueueFailureMessage(failedRequirements[i])
	}

	if status.HasFailureMessages() {
		status.End(cli.Failure)
		return false
	}

	status.EndWithSuccess("Kubernetes version %q is supported", k8sVersion)

	return true
}
