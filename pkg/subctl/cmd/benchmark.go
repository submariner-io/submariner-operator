package cmd

import (
	"fmt"

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
	msg := "Runs the test within a single cluster"
	benchmarkLatencyCmd.PersistentFlags().BoolVar(&intraCluster, "intra-cluster", false, msg)

	benchmarkThroughputCmd.PersistentFlags().BoolVar(&intraCluster, "intra-cluster", false, msg)

	benchmarkCmd.AddCommand(benchmarkThroughputCmd)
	benchmarkCmd.AddCommand(benchmarkLatencyCmd)
	rootCmd.AddCommand(benchmarkCmd)
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

	fmt.Printf("Performing throughput tests\n")
	benchmark.StartThroughputTests(intraCluster)
}

func testLatency(cmd *cobra.Command, args []string) {
	configureTestingFramework(args)

	fmt.Printf("Performing latency tests\n")
	benchmark.StartLatencyTests(intraCluster)
}
