/*
© 2021 Red Hat, Inc. and others.

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
	"github.com/spf13/cobra"
)

var (
	validateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate Submariner deployment and report any issues",
		Long: "This command validates a Submariner deployment and reports any issues if \n" +
			"1. Submariner prerequisites are missing on the cluster\n" +
			"2. Submariner is deployed on an unsupported K8s cluster\n" +
			"3. Submariner Pods are in an error state\n",
	}
	validateK8sCmd = &cobra.Command{
		Use:   "k8s",
		Short: "Validate if K8s cluster configuration is supported by Submariner",
		Long: "This command verifies if any unsupported K8s version, CNI," +
			" or kube-proxy mode is detected on the cluster.",
		Run: validateK8sConfig,
	}
	verboseVerification     bool
	supportedNetworkPlugins = []string{"generic", "canal-flannel", "weave-net", "OpenShiftSDN", "OVNKubernetes"}
)

func init() {
	addKubeconfigFlag(validateCmd)
	addCommonValidateFlags(validateCmd)
	validateCmd.AddCommand(validateK8sCmd)
	rootCmd.AddCommand(validateCmd)
}

func addCommonValidateFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&verboseVerification, "verbose", false,
		"produce verbose logs while validating the setup.")
	cmd.PersistentFlags().StringVar(&submarinerNamespace, "submariner-namespace", "submariner-operator",
		"namespace in which Submariner is deployed")
}
