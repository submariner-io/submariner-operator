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

package prepare

import (
	"github.com/spf13/cobra"
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/generic"
)

func newGenericPrepareCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generic",
		Short: "Prepares a generic cluster for Submariner",
		Long:  "This command labels the required number of gateway nodes for Submariner installation.",
		Run:   prepareGenericCluster,
	}

	cmd.Flags().IntVar(&gateways, "gateways", DefaultNumGateways,
		"Number of gateways to deploy")

	return cmd
}

func prepareGenericCluster(cmd *cobra.Command, args []string) {
	// nolint:wrapcheck // No need to wrap errors here.
	err := generic.RunOnK8sCluster(
		*parentRestConfigProducer, cli.NewReporter(),
		func(gwDeployer api.GatewayDeployer, status reporter.Interface) error {
			if gateways > 0 {
				gwInput := api.GatewayDeployInput{
					Gateways: gateways,
				}
				err := gwDeployer.Deploy(gwInput, status)
				if err != nil {
					return err
				}
			}

			return nil
		})

	exit.OnErrorWithMessage(err, "Failed to prepare generic K8s cluster")
}
