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

// nolint:revive // Blank imports below are intentional.
import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/AlecAivazis/survey/v2"
	"github.com/onsi/ginkgo/config"
	"github.com/spf13/cobra"
	_ "github.com/submariner-io/lighthouse/test/e2e/discovery"
	_ "github.com/submariner-io/lighthouse/test/e2e/framework"
	"github.com/submariner-io/shipyard/test/e2e"
	"github.com/submariner-io/shipyard/test/e2e/framework"
	"github.com/submariner-io/submariner-operator/internal/component"
	"github.com/submariner-io/submariner-operator/internal/exit"
	_ "github.com/submariner-io/submariner/test/e2e/dataplane"
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
	disruptiveTests                 bool
)

var verifyCmd = &cobra.Command{
	Use:   "verify --kubecontexts <kubeContext1>,<kubeContext2>",
	Short: "Run verifications between two clusters",
	Long: `This command performs various tests to verify that a Submariner deployment between two clusters
is functioning properly. The verifications performed are controlled by the --only and --enable-disruptive
flags. All verifications listed in --only are performed with special handling for those deemed as disruptive.
A disruptive verification is one that changes the state of the clusters as a side effect. If running the
command interactively, you will be prompted for confirmation to perform disruptive verifications unless
the --enable-disruptive flag is also specified. If running non-interactively (that is with no stdin),
--enable-disruptive must be specified otherwise disruptive verifications are skipped.

The following verifications are deemed disruptive:

    ` + strings.Join(disruptiveVerificationNames(), "\n    "),
	Args: func(cmd *cobra.Command, args []string) error {
		if err := checkValidateArguments(args); err != nil {
			return err
		}
		return checkVerifyArguments()
	},
	Run: func(cmd *cobra.Command, args []string) {
		testType := ""
		if len(args) > 0 {
			fmt.Println("subctl verify with kubeconfig arguments is deprecated, please use --kubecontexts instead")
		}
		err := setUpTestFramework(args, restConfigProducer)
		exit.OnErrorWithMessage(err, "error setting up test framework")

		disruptive := extractDisruptiveVerifications(verifyOnly)
		if !disruptiveTests && len(disruptive) > 0 {
			err := survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("You have specified disruptive verifications (%s). Are you sure you want to run them?",
					strings.Join(disruptive, ",")),
			}, &disruptiveTests)
			if err != nil {
				if isNonInteractive(err) {
					fmt.Printf(`
You have specified disruptive verifications (%s) but subctl is running non-interactively and thus cannot
prompt for confirmation therefore you must specify --enable-disruptive to run them.`, strings.Join(disruptive, ","))
				} else {
					exit.OnErrorWithMessage(err, "Prompt failure:")
				}
			}
		}

		patterns, verifications, err := getVerifyPatterns(verifyOnly, disruptiveTests)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		config.GinkgoConfig.FocusStrings = patterns

		fmt.Printf("Performing the following verifications: %s\n", strings.Join(verifications, ", "))

		if !e2e.RunE2ETests(&testing.T{}) {
			exit.WithMessage(fmt.Sprintf("[%s] E2E failed", testType))
		}
	},
}

func init() {
	restConfigProducer.AddKubeContextMultiFlag(verifyCmd, "comma-separated list of exactly two kubeconfig contexts to use.")
	addVerifyFlags(verifyCmd)
	rootCmd.AddCommand(verifyCmd)

	framework.AddBeforeSuite(detectGlobalnet)
}

func addVerifyFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&verboseConnectivityVerification, "verbose", false, "produce verbose logs during connectivity verification")
	cmd.Flags().UintVar(&operationTimeout, "operation-timeout", 240, "operation timeout for K8s API calls")
	cmd.Flags().UintVar(&connectionTimeout, "connection-timeout", 60, "timeout in seconds per connection attempt")
	cmd.Flags().UintVar(&connectionAttempts, "connection-attempts", 2, "maximum number of connection attempts")
	cmd.Flags().StringVar(&reportDirectory, "report-dir", ".", "XML report directory")
	cmd.Flags().StringVar(&submarinerNamespace, "submariner-namespace", "submariner-operator", "namespace in which submariner is deployed")
	cmd.Flags().StringVar(&verifyOnly, "only", strings.Join(getAllVerifyKeys(), ","), "comma separated verifications to be performed")
	cmd.Flags().BoolVar(&disruptiveTests, "disruptive-tests", false, "enable disruptive verifications like gateway-failover")
}

func isNonInteractive(err error) bool {
	if errors.Is(err, io.EOF) {
		return true
	}

	var pathError *os.PathError
	if errors.As(err, &pathError) {
		var syserr syscall.Errno
		if errors.As(pathError, &syserr) {
			if pathError.Path == "/dev/stdin" && (errors.Is(syserr, syscall.EBADF) || errors.Is(syserr, syscall.EINVAL)) {
				return true
			}
		}
	}

	return false
}

func checkValidateArguments(args []string) error {
	if len(args) != 2 && restConfigProducer.CountRequestedClusters() != 2 {
		return fmt.Errorf("two kubecontexts must be specified")
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

	if connectionAttempts < 1 {
		return fmt.Errorf("--connection-attempts must be >=1")
	}

	if connectionTimeout < 20 {
		return fmt.Errorf("--connection-timeout must be >=20")
	}

	return nil
}

func checkVerifyArguments() error {
	if _, _, err := getVerifyPatterns(verifyOnly, true); err != nil {
		return err
	}

	return nil
}

var verifyE2EPatterns = map[string]string{
	component.Connectivity:     "\\[dataplane",
	component.ServiceDiscovery: "\\[discovery",
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

func disruptiveVerificationNames() []string {
	names := make([]string, 0, len(verifyE2EDisruptivePatterns))
	for n := range verifyE2EDisruptivePatterns {
		names = append(names, n)
	}

	return names
}

func extractDisruptiveVerifications(csv string) []string {
	var disruptive []string

	verifications := strings.Split(csv, ",")
	for _, verification := range verifications {
		verification = strings.Trim(strings.ToLower(verification), " ")
		if _, ok := verifyE2EDisruptivePatterns[verification]; ok {
			disruptive = append(disruptive, verification)
		}
	}

	return disruptive
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

func getVerifyPatterns(csv string, includeDisruptive bool) ([]string, []string, error) {
	outputPatterns := []string{}
	outputVerifications := []string{}

	verifications := strings.Split(csv, ",")
	for _, verification := range verifications {
		verification = strings.Trim(strings.ToLower(verification), " ")

		vtype, pattern := getVerifyPattern(verification)
		switch vtype {
		case unknownVerification:
			return nil, nil, fmt.Errorf("unknown verification %q", verification)
		case normalVerification:
			outputPatterns = append(outputPatterns, pattern)
			outputVerifications = append(outputVerifications, verification)
		case disruptiveVerification:
			if includeDisruptive {
				outputPatterns = append(outputPatterns, pattern)
				outputVerifications = append(outputVerifications, verification)
			}
		}
	}

	if len(outputPatterns) == 0 {
		return nil, nil, fmt.Errorf("please specify at least one verification to be performed")
	}

	return outputPatterns, outputVerifications, nil
}
