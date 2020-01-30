/*
© 2019 Red Hat, Inc. and others.

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
	"testing"

	"github.com/onsi/ginkgo/config"
	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner/test/e2e"
	_ "github.com/submariner-io/submariner/test/e2e/dataplane"
	"github.com/submariner-io/submariner/test/e2e/framework"
)

func init() {
	rootCmd.AddCommand(validateCmd)

}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate connectivity between two clusters",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		err := checkValidateArguments(args)
		exitOnError("Argument missing", err)

		framework.TestContext.KubeConfigs = args
		framework.TestContext.OperationTimeout = 240
		framework.TestContext.ConnectionTimeout = 60
		framework.TestContext.ConnectionAttempts = 5
		framework.TestContext.ReportDir = "."
		framework.TestContext.ReportPrefix = "subctl"
		// For some tests this is only printing, but in some of them they need those to be
		// the cluster IDs that will be registered in the Cluster CRDs by submariner
		framework.TestContext.ClusterIDs = []string{"ClusterA", "ClusterB"}
		config.GinkgoConfig.FocusString = "dataplane"
		t := testing.T{}
		e2e.RunE2ETests(&t)
	},
}

func checkValidateArguments(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("validate needs exactly two kubeconfigs to validate connectivity between two clusters")
	}
	return nil
}
