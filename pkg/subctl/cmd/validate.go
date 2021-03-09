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
			"1. Submariner pre-requisites are missing on the cluster\n" +
			"2. Submariner is deployed on an unsupported k8s cluster\n" +
			"3. Submariner Pods are in an error state\n",
		Run: validateSubmariner,
	}
	validateK8sCmd = &cobra.Command{
		Use:   "k8s",
		Short: "Validate if K8s Cluster configuration is supported by Submariner",
		Long: "This command verifies if any unsupported configuration like K8s version, CNI," +
			" kube-proxy mode is detected on the cluster.",
	}
	verboseVerification bool
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
		"namespace in which submariner is deployed")
}
