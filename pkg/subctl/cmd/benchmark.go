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
		Short: "Benchmark features between two clusters",
		Long:  "This command used to run various benchmark tests between two clusters",
	}
	benchmarkThroughputCmd = &cobra.Command{
		Use:   "throughput <kubeconfig1> [<kubeconfig2>]",
		Short: "Benchmark throughput between two clusters",
		Long:  "This command runs throughput tests between two clusters",
		Args: func(cmd *cobra.Command, args []string) error {
			return checkThroughputArguments(args, intraCluster)
		},
		Run: testThroughput,
	}
	benchmarkLatencyCmd = &cobra.Command{
		Use:   "latency <kubeconfig1> [<kubeconfig2>]",
		Short: "Benchmark latency between two clusters",
		Long:  "This command runs latency benchmark tests between two clusters",
		Args: func(cmd *cobra.Command, args []string) error {
			return checkThroughputArguments(args, intraCluster)
		},
		Run: testLatency,
	}
)

func init() {
	benchmarkLatencyCmd.PersistentFlags().BoolVar(&intraCluster, "intra-cluster-baseline", false,
		"Run benchmark latency test with intra-cluster pods")

	benchmarkThroughputCmd.PersistentFlags().BoolVar(&intraCluster, "intra-cluster-baseline", false,
		"Run benchmark latency test with intra-cluster pods")

	benchmarkCmd.AddCommand(benchmarkThroughputCmd)
	benchmarkCmd.AddCommand(benchmarkLatencyCmd)
	rootCmd.AddCommand(benchmarkCmd)
}

func checkThroughputArguments(args []string, intraCluster bool) error {
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
