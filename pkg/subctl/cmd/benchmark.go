package cmd

import (
	"fmt"
	"testing"

	"github.com/onsi/ginkgo/config"
	"github.com/submariner-io/shipyard/test/e2e"
	_ "github.com/submariner-io/submariner-operator/pkg/subctl/benchmark"

	"github.com/spf13/cobra"
)

var (
	benchmarkCmd = &cobra.Command{
		Use:   "benchmark",
		Short: "Benchmark features between two clusters",
		Long:  "This command used to benchmark variety features between two clusters",
	}
	benchmarkThroughputCmd = &cobra.Command{
		Use:   "throughput <kubeconfig1> <kubeconfig2>",
		Short: "Benchmark throughput between two clusters",
		Long:  "This command benchmark the throughput performance between two clusters",
		Args: func(cmd *cobra.Command, args []string) error {
			return checkThroughputArguments(args)
		},
		Run: testThroughput,
	}
)

func init() {
	benchmarkCmd.AddCommand(benchmarkThroughputCmd)
	rootCmd.AddCommand(benchmarkCmd)
}

func checkThroughputArguments(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("Two kubeconfigs must be specified.")
	}
	return nil
}

func testThroughput(cmd *cobra.Command, args []string) {
	testType := ""
	verboseConnectivityVerification = true
	configureTestingFramework(args)

	config.GinkgoConfig.FocusString = "\\[throughput"
	fmt.Printf("Performing throughput tests\n")
	if !e2e.RunE2ETests(&testing.T{}) {
		exitWithErrorMsg(fmt.Sprintf("[%s] E2E failed", testType))
	}
}
