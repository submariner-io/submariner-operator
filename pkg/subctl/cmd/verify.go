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
	"github.com/submariner-io/shipyard/test/e2e/framework"

	verifyconnectivity "github.com/submariner-io/submariner-operator/pkg/subctl/verify/connectivity"
	verifyservicediscovery "github.com/submariner-io/submariner-operator/pkg/subctl/verify/servicediscovery"
)

var (
	verifyConnectivity              bool
	verifyServiceDiscovery          bool
	verifyAll                       bool
	verboseConnectivityVerification bool
	operationTimeout                uint
	connectionTimeout               uint
	connectionAttempts              uint
	reportDirectory                 string
	submarinerNamespace             string
)

func init() {
	verifyCmd.Flags().BoolVar(&verifyConnectivity, "connectivity", true, "verify connectivity between two clusters")
	verifyCmd.Flags().BoolVar(&verifyServiceDiscovery, "service-discovery", false, "verify service-discovery between two clusters")
	verifyCmd.Flags().BoolVar(&verifyAll, "all", false, "verify connectivity and service-discovery between two clusters")
	addVerifyFlags(verifyCmd)
	rootCmd.AddCommand(verifyCmd)
}

func addVerifyFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&verboseConnectivityVerification, "verbose", false, "Produce verbose logs during connectivity verification")
	cmd.Flags().UintVar(&operationTimeout, "operation-timeout", 240, "Operation timeout for K8s API calls")
	cmd.Flags().UintVar(&connectionTimeout, "connection-timeout", 60, "The timeout in seconds per connection attempt ")
	cmd.Flags().UintVar(&connectionAttempts, "connection-attempts", 2, "The maximum number of connection attempts")
	cmd.Flags().StringVar(&reportDirectory, "report-dir", ".", "XML report directory")
	cmd.Flags().StringVar(&submarinerNamespace, "submariner-namespace", "submariner-operator", "Namespace in which submariner is deployed")
}

var verifyCmd = &cobra.Command{
	Use:   "verify <kubeConfig1> <kubeConfig2>",
	Short: "Verify connectivity/service-discovery between two clusters",
	Args: func(cmd *cobra.Command, args []string) error {
		return checkValidateArguments(args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		configureTestingFramework(args)
		if verifyConnectivity || verifyAll {
			verifyconnectivity.RunE2E(&testing.T{})
		}
		if verifyServiceDiscovery || verifyAll {
			config.GinkgoConfig.FocusString = "discovery"
			verifyservicediscovery.RunE2E(&testing.T{})
		}
	},
}

func configureTestingFramework(args []string) {
	framework.TestContext.KubeConfig = ""
	framework.TestContext.KubeConfigs = args
	framework.TestContext.OperationTimeout = operationTimeout
	framework.TestContext.ConnectionTimeout = connectionTimeout
	framework.TestContext.ConnectionAttempts = connectionAttempts
	framework.TestContext.ReportDir = reportDirectory
	framework.TestContext.ReportPrefix = "subctl"
	framework.TestContext.SubmarinerNamespace = submarinerNamespace

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
