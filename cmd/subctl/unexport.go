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

package subctl

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/pkg/service"
	mcsclient "sigs.k8s.io/mcs-api/pkg/client/clientset/versioned/typed/apis/v1alpha1"
)

var (
	unexportOptions struct {
		namespace string
	}
	unexportCmd = &cobra.Command{
		Use:   "unexport",
		Short: "Stop a resource from being exported to other clusters",
		Long:  "This command stops exporting a resource so that it's no longer accessible to other clusters",
	}
	unexportServiceCmd = &cobra.Command{
		Use:   "service <serviceName>",
		Short: "Stop a Service from being exported to other clusters",
		Long: "This command removes the ServiceExport resource with the given name which in turn stops the Service " +
			"of the same name from being exported to other clusters",
		PreRunE: restConfigProducer.CheckVersionMismatch,
		Run: func(cmd *cobra.Command, args []string) {
			err := validateUnexportArguments(args)
			exit.OnErrorWithMessage(err, "Insufficient arguments")

			status := cli.NewReporter()

			config, err := restConfigProducer.ForCluster()
			exit.OnError(status.Error(err, "Error creating REST config"))

			clientConfig := restConfigProducer.ClientConfig()
			if unexportOptions.namespace == "" {
				if unexportOptions.namespace, _, err = clientConfig.Namespace(); err != nil {
					unexportOptions.namespace = "default"
				}
			}

			client, err := mcsclient.NewForConfig(config.Config)
			exit.OnError(status.Error(err, "Error creating client"))

			err = service.Unexport(client, unexportOptions.namespace, args[0], status)
			exit.OnError(err)
		},
	}
)

func init() {
	restConfigProducer.AddKubeContextFlag(unexportCmd)
	unexportServiceCmd.Flags().StringVarP(&unexportOptions.namespace, "namespace", "n", "", "namespace of the service to be unexported")
	unexportCmd.AddCommand(unexportServiceCmd)
	rootCmd.AddCommand(unexportCmd)
}

func validateUnexportArguments(args []string) error {
	if len(args) == 0 || args[0] == "" {
		return errors.New("name of the Service to be removed must be specified")
	}

	return nil
}
