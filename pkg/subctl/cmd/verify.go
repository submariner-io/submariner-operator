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

package cmd

import (
	"context"
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
	_ "github.com/submariner-io/submariner/test/e2e/dataplane"
	_ "github.com/submariner-io/submariner/test/e2e/redundancy"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/subctl/components"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
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

func init() {
	addKubeContextMultiFlag(verifyCmd)
	verifyCmd.Flags().StringVar(&verifyOnly, "only", strings.Join(getAllVerifyKeys(), ","), "comma separated verifications to be performed")
	verifyCmd.Flags().BoolVar(&disruptiveTests, "disruptive-tests", false, "enable disruptive verifications like gateway-failover")
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
}

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
		err := configureTestingFramework(args)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

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
					exitWithErrorMsg(fmt.Sprintf("Prompt failure: %#v", err))
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
			exitWithErrorMsg(fmt.Sprintf("[%s] E2E failed", testType))
		}
	},
}

func isNonInteractive(err error) bool {
	if err == io.EOF {
		return true
	}

	if pathError, ok := err.(*os.PathError); ok {
		if syserr, ok := pathError.Err.(syscall.Errno); ok {
			if pathError.Path == "/dev/stdin" && (syserr == syscall.EBADF || syserr == syscall.EINVAL) {
				return true
			}
		}
	}

	return false
}

func configureTestingFramework(args []string) error {
	// Legacy handling: if arguments are provided and are files, assume they are kubeconfigs;
	// otherwise, use contexts from --kubecontexts
	// This is shared by verify and benchmark
	if len(args) > 0 {
		_, err1 := os.Stat(args[0])
		var err2 error = nil
		if len(args) > 1 {
			_, err2 = os.Stat(args[1])
		}
		if err1 != nil || err2 != nil {
			// Something happened (possibly IsNotExist, but we don’t care about specifics)
			return fmt.Errorf("the provided arguments (%v) aren't accessible files", args)
		}

		// The files exist and can be examined without error
		framework.TestContext.KubeConfig = ""
		framework.TestContext.KubeConfigs = args

		// Read the cluster names from the given kubeconfigs
		for _, config := range args {
			framework.TestContext.ClusterIDs = append(framework.TestContext.ClusterIDs, clusterNameFromConfig(config, ""))
		}
	} else {
		framework.TestContext.KubeContexts = kubeContexts
	}
	framework.TestContext.OperationTimeout = operationTimeout
	framework.TestContext.ConnectionTimeout = connectionTimeout
	framework.TestContext.ConnectionAttempts = connectionAttempts
	framework.TestContext.ReportDir = reportDirectory
	framework.TestContext.ReportPrefix = "subctl"
	framework.TestContext.SubmarinerNamespace = submarinerNamespace

	config.DefaultReporterConfig.Verbose = verboseConnectivityVerification
	config.DefaultReporterConfig.SlowSpecThreshold = 60

	return nil
}

func clusterNameFromConfig(kubeConfigPath, kubeContext string) string {
	rawConfig, err := getClientConfig(kubeConfigPath, "").RawConfig()
	exitOnError(fmt.Sprintf("Error obtaining the kube config for path %q", kubeConfigPath), err)
	cluster := getClusterNameFromContext(rawConfig, kubeContext)
	if cluster == nil {
		exitWithErrorMsg(fmt.Sprintf("Could not obtain the cluster name from kube config: %#v", rawConfig))
	}

	return *cluster
}

func checkValidateArguments(args []string) error {
	if len(args) != 2 && len(kubeContexts) != 2 {
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
	} else if strings.Compare(kubeContexts[0], kubeContexts[1]) == 0 {
		return fmt.Errorf("the two kubecontexts must be different")
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
	components.Connectivity:     "\\[dataplane",
	components.ServiceDiscovery: "\\[discovery",
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
	var names = make([]string, 0, len(verifyE2EDisruptivePatterns))
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

func detectGlobalnet() {
	submarinerClient, err := submarinerclientset.NewForConfig(framework.RestConfigs[framework.ClusterA])
	exitOnError("Error creating submariner client: %v", err)

	submariner, err := submarinerClient.SubmarinerV1alpha1().Submariners(OperatorNamespace).Get(
		context.TODO(), submarinercr.SubmarinerName, v1.GetOptions{})
	if errors.IsNotFound(err) {
		exitWithErrorMsg(`
The Submariner resource was not found. Either submariner has not been deployed in this cluster or was deployed using helm.
This command only supports submariner deployed using the operator via 'subctl join'.`)
	}

	if err != nil {
		exitOnError("Error obtaining Submariner resource: %v", err)
	}

	framework.TestContext.GlobalnetEnabled = submariner.Spec.GlobalCIDR != ""
}
