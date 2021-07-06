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
	autil "github.com/submariner-io/admiral/pkg/util"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils/restconfig"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	mcsv1a1 "sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

var (
	exportCmd = &cobra.Command{
		Use:   "export",
		Short: "Exports a resource to other clusters",
		Long:  "This command exports a resource so it is accessible to other clusters",
	}
	exportServiceCmd = &cobra.Command{
		Use:   "service <serviceName>",
		Short: "Exports a Service to other clusters",
		Long: "This command creates a ServiceExport resource with the given name which causes the Service of the same name to be accessible" +
			" to other clusters",
		PreRunE: CheckVersionMismatch,
		Run:     exportService,
	}
	serviceNamespace string
)

func init() {
	AddKubeConfigFlag(exportCmd)
	addServiceExportFlags(exportServiceCmd)
	exportCmd.AddCommand(exportServiceCmd)
	rootCmd.AddCommand(exportCmd)
}

func addServiceExportFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&serviceNamespace, "namespace", "n", "", "namespace of the service to be exported")
}

func exportService(cmd *cobra.Command, args []string) {
	err := validateArguments(args)
	utils.ExitOnError("Insufficient arguments", err)

	clientConfig := restconfig.ClientConfig(kubeConfig, kubeContext)
	restConfig, err := clientConfig.ClientConfig()

	utils.ExitOnError("Error connecting to the target cluster", err)

	dynClient, clientSet, err := restconfig.Clients(restConfig)
	utils.ExitOnError("Error connecting to the target cluster", err)
	restMapper, err := autil.BuildRestMapper(restConfig)

	utils.ExitOnError("Error creating RestMapper for ServiceExport", err)
	if serviceNamespace == "" {
		if serviceNamespace, _, err = clientConfig.Namespace(); err != nil {
			serviceNamespace = "default"
		}
	}

	err = mcsv1a1.AddToScheme(scheme.Scheme)
	utils.ExitOnError("Failed to add to scheme", err)

	svcName := args[0]

	_, err = clientSet.CoreV1().Services(serviceNamespace).Get(context.TODO(), svcName, metav1.GetOptions{})
	utils.ExitOnError(fmt.Sprintf("Unable to find the Service %q in namespace %q", svcName, serviceNamespace), err)

	mcsServiceExport := &mcsv1a1.ServiceExport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: serviceNamespace,
		},
	}
	resourceServiceExport, gvr, err := autil.ToUnstructuredResource(mcsServiceExport, restMapper)
	utils.ExitOnError("Failed to convert to Unstructured", err)

	_, err = dynClient.Resource(*gvr).Namespace(serviceNamespace).Create(context.TODO(), resourceServiceExport, metav1.CreateOptions{})
	if k8serrors.IsAlreadyExists(err) {
		fmt.Fprintln(os.Stdout, "Service already exported")
		return
	}
	utils.ExitOnError("Failed to export Service", err)
	fmt.Fprintln(os.Stdout, "Service exported successfully")
}

func validateArguments(args []string) error {
	if len(args) == 0 || args[0] == "" {
		return errors.New("name of the Service to be exported must be specified")
	}
	return nil
}
