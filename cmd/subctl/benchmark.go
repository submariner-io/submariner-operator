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
	"fmt"
	"strings"

	"github.com/onsi/ginkgo/config"
	"github.com/spf13/cobra"
	"github.com/submariner-io/shipyard/test/e2e/framework"
	"github.com/submariner-io/submariner-operator/internal/benchmark"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
)

var (
	intraCluster bool
	verbose      bool

	benchmarkCmd = &cobra.Command{
		Use:   "benchmark",
		Short: "Benchmark tests",
		Long:  "This command runs various benchmark tests",
	}
	benchmarkThroughputCmd = &cobra.Command{
		Use:   "throughput --kubecontexts <kubeContext1>[,<kubeContext2>]",
		Short: "Benchmark throughput",
		Long:  "This command runs throughput tests within a cluster or between two clusters",
		Args: func(cmd *cobra.Command, args []string) error {
			return checkBenchmarkArguments(args, intraCluster)
		},
		Run: func(command *cobra.Command, args []string) {
			err := setUpTestFramework(args, restConfigProducer)
			exit.OnErrorWithMessage(err, "error setting up test framework")
			benchmark.StartThroughputTests(intraCluster, verbose)
		},
	}
	benchmarkLatencyCmd = &cobra.Command{
		Use:   "latency --kubecontexts <kubeContext1>[,<kubeContext2>]",
		Short: "Benchmark latency",
		Long:  "This command runs latency benchmark tests within a cluster or between two clusters",
		Args: func(cmd *cobra.Command, args []string) error {
			return checkBenchmarkArguments(args, intraCluster)
		},
		Run: func(command *cobra.Command, args []string) {
			err := setUpTestFramework(args, restConfigProducer)
			exit.OnErrorWithMessage(err, "error setting up test framework")
			benchmark.StartLatencyTests(intraCluster, verbose)
		},
	}
)

func init() {
	addBenchmarkFlags(benchmarkLatencyCmd)
	addBenchmarkFlags(benchmarkThroughputCmd)

	benchmarkCmd.AddCommand(benchmarkThroughputCmd)
	benchmarkCmd.AddCommand(benchmarkLatencyCmd)
	rootCmd.AddCommand(benchmarkCmd)

	framework.AddBeforeSuite(detectGlobalnet)
}

func addBenchmarkFlags(cmd *cobra.Command) {
	restConfigProducer.AddKubeContextMultiFlag(cmd, "comma-separated list of one or two kubeconfig contexts to use.")
	cmd.PersistentFlags().BoolVar(&intraCluster, "intra-cluster", false, "run the test within a single cluster")
	cmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "produce verbose logs during benchmark tests")
}

func checkBenchmarkArguments(args []string, intraCluster bool) error {
	if !intraCluster && len(args) != 2 && restConfigProducer.CountRequestedClusters() != 2 {
		return fmt.Errorf("two kubecontexts must be specified")
	} else if intraCluster && len(args) != 1 && restConfigProducer.CountRequestedClusters() != 1 {
		return fmt.Errorf("only one kubecontext should be specified")
	}

	if len(args) == 2 {
		if strings.Compare(args[0], args[1]) == 0 {
			return fmt.Errorf("kubeconfig file <kubeConfig1> and <kubeConfig2> cannot be the same file")
		}

		same, err := compareFiles(args[0], args[1])
		if err != nil {
			return err
		}

		if same {
			return fmt.Errorf("kubeconfig file <kubeConfig1> and <kubeConfig2> need to have a unique content")
		}
	}

	return nil
}

func setUpTestFramework(args []string, restConfigProducer restconfig.Producer) error {
	if len(args) > 0 {
		err := restconfig.ConfigureTestFramework(args)
		if err != nil {
			return err //nolint:wrapcheck // error can't be wrapped
		}
	} else {
		restConfigProducer.PopulateTestFramework()
	}

	framework.TestContext.OperationTimeout = operationTimeout
	framework.TestContext.ConnectionTimeout = connectionTimeout
	framework.TestContext.ConnectionAttempts = connectionAttempts
	framework.TestContext.JunitReport = junitReport
	framework.TestContext.SubmarinerNamespace = constants.SubmarinerNamespace

	config.DefaultReporterConfig.Verbose = verboseConnectivityVerification
	config.DefaultReporterConfig.SlowSpecThreshold = 60

	return nil
}
