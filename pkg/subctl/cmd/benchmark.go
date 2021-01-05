/*
© 2021 Red Hat, Inc. and others

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

	"github.com/submariner-io/shipyard/test/e2e/framework"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/subctl/benchmark"
)

var (
	intraCluster bool

	benchmarkCmd = &cobra.Command{
		Use:   "benchmark",
		Short: "Benchmark tests",
		Long:  "This command runs various benchmark tests",
	}
	benchmarkThroughputCmd = &cobra.Command{
		Use:   "throughput <kubeconfig1> [<kubeconfig2>]",
		Short: "Benchmark throughput",
		Long:  "This command runs throughput tests within a cluster or between two clusters",
		Args: func(cmd *cobra.Command, args []string) error {
			return checkBenchmarkArguments(args, intraCluster)
		},
		Run: testThroughput,
	}
	benchmarkLatencyCmd = &cobra.Command{
		Use:   "latency <kubeconfig1> [<kubeconfig2>]",
		Short: "Benchmark latency",
		Long:  "This command runs latency benchmark tests within a cluster or between two clusters",
		Args: func(cmd *cobra.Command, args []string) error {
			return checkBenchmarkArguments(args, intraCluster)
		},
		Run: testLatency,
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
	cmd.PersistentFlags().BoolVar(&intraCluster, "intra-cluster", false, "run the test within a single cluster")
	cmd.PersistentFlags().BoolVar(&benchmark.Verbose, "verbose", false, "produce verbose logs during benchmark tests")
}

func checkBenchmarkArguments(args []string, intraCluster bool) error {
	if !intraCluster && len(args) != 2 {
		return fmt.Errorf("Two kubeconfigs must be specified.")
	} else if intraCluster && len(args) != 1 {
		return fmt.Errorf("Only one kubeconfig should be specified.")
	}
	return nil
}

func testThroughput(cmd *cobra.Command, args []string) {
	configureTestingFramework(args)

	if benchmark.Verbose {
		fmt.Printf("Performing throughput tests\n")
	}
	benchmark.StartThroughputTests(intraCluster)
}

func testLatency(cmd *cobra.Command, args []string) {
	configureTestingFramework(args)

	if benchmark.Verbose {
		fmt.Printf("Performing latency tests\n")
	}
	benchmark.StartLatencyTests(intraCluster)
}
