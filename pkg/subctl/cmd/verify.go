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
	"io"
	"strings"
	"testing"

	"github.com/AlecAivazis/survey/v2"
	"github.com/onsi/ginkgo/config"
	"github.com/spf13/cobra"
	_ "github.com/submariner-io/lighthouse/test/e2e/discovery"
	_ "github.com/submariner-io/lighthouse/test/e2e/framework"
	"github.com/submariner-io/shipyard/test/e2e"
	"github.com/submariner-io/shipyard/test/e2e/framework"
	_ "github.com/submariner-io/submariner/test/e2e/dataplane"
	_ "github.com/submariner-io/submariner/test/e2e/framework"
	_ "github.com/submariner-io/submariner/test/e2e/redundancy"
)

var (
	verboseConnectivityVerification bool
	operationTimeout                uint
	connectionTimeout               uint
	connectionAttempts              uint
	reportDirectory                 string
	submarinerNamespace             string
	verifyOnly                      string
	enableDisruptive                bool
)

func init() {
	verifyCmd.Flags().StringVar(&verifyOnly, "only", strings.Join(getAllVerifyKeys(), ","), "comma separated verifications to be performed")
	verifyCmd.Flags().BoolVar(&enableDisruptive, "enable-disruptive", false, "enable disruptive verifications like gateway-failover")
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
	Short: "Run verifications between two clusters",
	Long: `The verify command is controlled by the --verify-only and --enable-disruptive flags,
all verifications in --enable-only will be executed, but the disruptive ones, like gateway
failover testing will be excluded filtered out unless --enable-disruptive is provided. If in
interactive mode the user will be asked about running disruptive testing.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := checkValidateArguments(args); err != nil {
			return err
		}
		return checkVerifyArguments()
	},
	Run: func(cmd *cobra.Command, args []string) {
		testType := ""
		configureTestingFramework(args)

		if !enableDisruptive && hasDisruptiveVerification(verifyOnly) {
			err := survey.AskOne(&survey.Confirm{
				Message: "Do you want to perform disruptive verifications like gateway-failover?",
			}, &enableDisruptive)
			if err == io.EOF {
				fmt.Println(`
subctl is running non-interactively, disruptive verifications are disabled, please use
--enable-disruptive if you want to run disruptive verifications, like gateway-failover`)
			}
		}

		patterns, verifications, _ := getVerifyPatterns(verifyOnly)
		config.GinkgoConfig.FocusString = strings.Join(patterns, "|")

		fmt.Printf("Performing the following verifications: %s\n", strings.Join(verifications, ", "))

		if !e2e.RunE2ETests(&testing.T{}) {
			exitWithErrorMsg(fmt.Sprintf("[%s] E2E failed", testType))
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

func checkVerifyArguments() error {
	if _, _, err := getVerifyPatterns(verifyOnly); err != nil {
		return err
	}
	return nil
}

var verifyE2EPatterns = map[string]string{
	"connectivity":      "\\[dataplane",
	"service-discovery": "\\[discovery",
}

var verifyE2EDisruptivePatterns = map[string]string{
	"gateway-failover": "\\[redundancy",
}

type verificationType int

const (
	disruptiveVerification = iota
	normalVerification
	unknownVerification
)

func hasDisruptiveVerification(csv string) bool {
	verifications := strings.Split(csv, ",")
	for _, verification := range verifications {
		verification = strings.Trim(strings.ToLower(verification), " ")
		if _, ok := verifyE2EDisruptivePatterns[verification]; ok {
			return true
		}
	}
	return false
}

func getAllVerifyKeys() []string {
	keys := []string{}

	for k := range verifyE2EPatterns {
		keys = append(keys, k)
	}
	for k := range verifyE2EDisruptivePatterns {
		keys = append(keys, k)
	}
	return keys
}

func getVerifyPattern(key string) (verificationType, string) {
	if pattern, ok := verifyE2EPatterns[key]; ok {
		return normalVerification, pattern
	}
	if pattern, ok := verifyE2EDisruptivePatterns[key]; ok {
		return disruptiveVerification, pattern
	}
	return unknownVerification, ""
}

func getVerifyPatterns(csv string) ([]string, []string, error) {

	outputPatterns := []string{}
	outputVerifications := []string{}

	verifications := strings.Split(csv, ",")
	for _, verification := range verifications {
		verification = strings.Trim(strings.ToLower(verification), " ")
		vtype, pattern := getVerifyPattern(verification)
		switch vtype {
		case unknownVerification:
			return nil, nil, fmt.Errorf("unknown verification pattern: %s", pattern)
		case normalVerification:
			outputPatterns = append(outputPatterns, pattern)
			outputVerifications = append(outputVerifications, verification)
		case disruptiveVerification:
			if enableDisruptive {
				outputPatterns = append(outputPatterns, pattern)
				outputVerifications = append(outputVerifications, verification)
			}
		}
	}

	if len(outputPatterns) == 0 {
		return nil, nil, fmt.Errorf("No verification to be performed, try --enable-disruptive")
	}
	return outputPatterns, outputVerifications, nil
}
