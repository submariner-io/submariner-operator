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
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/exit"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	mcsclient "sigs.k8s.io/mcs-api/pkg/client/clientset/versioned/typed/apis/v1alpha1"
)

var (
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
		Run:     unexportService,
	}
)

func init() {
	restConfigProducer.AddKubeContextFlag(unexportCmd)
	addNamespaceFlag(unexportServiceCmd)
	unexportCmd.AddCommand(unexportServiceCmd)
	rootCmd.AddCommand(unexportCmd)
}

func unexportService(cmd *cobra.Command, args []string) {
	err := validateUnexportArguments(args)
	exit.OnErrorWithMessage(err, "Insufficient arguments")

	clientConfig := restConfigProducer.ClientConfig()
	restConfig, err := clientConfig.ClientConfig()
	exit.OnErrorWithMessage(err, "Error connecting to the target cluster")

	client, err := mcsclient.NewForConfig(restConfig)
	exit.OnErrorWithMessage(err, "Error connecting to the target cluster")

	if namespace == "" {
		if namespace, _, err = clientConfig.Namespace(); err != nil {
			namespace = "default"
		}
	}

	err = client.ServiceExports(namespace).Delete(context.TODO(), args[0], metav1.DeleteOptions{})

	exit.OnErrorWithMessage(err, "Failed to unexport Service")
	fmt.Fprintln(os.Stdout, "Service unexported successfully")
}

func validateUnexportArguments(args []string) error {
	if len(args) == 0 || args[0] == "" {
		return errors.New("name of the Service to be removed must be specified")
	}

	return nil
}
