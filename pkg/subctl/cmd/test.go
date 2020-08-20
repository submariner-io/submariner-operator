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
	testCmd = &cobra.Command{
		Use:   "test",
		Short: "Test features between two clusters",
		Long:  "This command used to test variety features between two clusters",
	}
	testThroughputCmd = &cobra.Command{
		Use:   "throughput <kubeconfig1> <kubeconfig2>",
		Short: "Test throughput between two clusters",
		Long:  "This command test the throughput performance between two clusters",
		Args: func(cmd *cobra.Command, args []string) error {
			return checkThroughputArguments(args)
		},
		Run: testThroughput,
	}
)

func init() {
	testCmd.AddCommand(testThroughputCmd)
	rootCmd.AddCommand(testCmd)
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
