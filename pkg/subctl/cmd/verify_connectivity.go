/*
© 2020 Red Hat, Inc. and others.

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

var (
	verboseConnectivityVerification bool
	operationTimeout                uint
	connectionTimeout               uint
	connectionAttempts              uint
	reportDirectory                 string
)

func init() {
	verifyConnectivityCmd.Flags().BoolVar(&verboseConnectivityVerification, "verbose", false, "Produce verbose logs during connectivity verification")
	verifyConnectivityCmd.Flags().UintVar(&operationTimeout, "operation-timeout", 240, "Operation timeout for K8s API calls")
	verifyConnectivityCmd.Flags().UintVar(&connectionTimeout, "connection-timeout", 60, "The timeout in seconds per connection attempt ")
	verifyConnectivityCmd.Flags().UintVar(&connectionAttempts, "connection-attempts", 2, "The maximum number of connection attempts")
	verifyConnectivityCmd.Flags().StringVar(&reportDirectory, "report-dir", ".", "XML report directory")

	rootCmd.AddCommand(verifyConnectivityCmd)
}

var verifyConnectivityCmd = &cobra.Command{
	Use:   "verify-connectivity <kubeConfig1> <kubeConfig2>",
	Short: "Verify connectivity between two clusters",
	Args: func(cmd *cobra.Command, args []string) error {
		return checkValidateArguments(args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		configureTestingFramework(args)
		e2e.RunE2ETests(&testing.T{})
	},
}

func configureTestingFramework(args []string) {
	framework.TestContext.KubeConfigs = args
	framework.TestContext.OperationTimeout = operationTimeout
	framework.TestContext.ConnectionTimeout = connectionTimeout
	framework.TestContext.ConnectionAttempts = connectionAttempts
	framework.TestContext.ReportDir = reportDirectory
	framework.TestContext.ReportPrefix = "subctl"

	// For some tests this is only printing, but in some of them they need those to be
	// the cluster IDs that will be registered in the Cluster CRDs by submariner
	framework.TestContext.ClusterIDs = []string{"ClusterA", "ClusterB"}

	config.GinkgoConfig.FocusString = "dataplane"
	config.DefaultReporterConfig.Verbose = verboseConnectivityVerification
	config.DefaultReporterConfig.SlowSpecThreshold = 60
}

func checkValidateArguments(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("Two kubeconfigs must be specified.")
	}
	if connectionAttempts < 1 {
		return fmt.Errorf("--connection-attempts must be >=1")
	}

	if connectionTimeout < 60 {
		return fmt.Errorf("--connection-timeout must be >=60")
	}
	return nil
}
