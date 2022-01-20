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
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/submariner-io/shipyard/test/e2e/framework"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/client"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var restConfigProducer = restconfig.NewProducer()

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "subctl",
	Short: "An installer for Submariner",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func detectGlobalnet() {
	clientProducer, err := client.NewProducerFromRestConfig(framework.RestConfigs[framework.ClusterA])
	exit.OnErrorWithMessage(err, "Error creating client producer")

	operatorClient := clientProducer.ForOperator()

	submariner, err := operatorClient.SubmarinerV1alpha1().Submariners(constants.OperatorNamespace).Get(
		context.TODO(), constants.SubmarinerName, v1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		exit.WithMessage("The Submariner resource was not found. Either submariner has not" +
			"been deployed in this cluster or was deployed using helm. This command only supports submariner deployed" +
			" using the operator via 'subctl join'.")
	}

	exit.OnErrorWithMessage(err, "Error obtaining Submariner resource")

	framework.TestContext.GlobalnetEnabled = submariner.Spec.GlobalCIDR != ""
}

func compareFiles(file1, file2 string) (bool, error) {
	first, err := os.ReadFile(file1)
	if err != nil {
		return false, errors.Wrapf(err, "error reading file %q", file1)
	}

	second, err := os.ReadFile(file2)
	if err != nil {
		return false, errors.Wrapf(err, "error reading file %q", file2)
	}

	return bytes.Equal(first, second), nil
}
