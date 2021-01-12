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
	"testing"

	"github.com/spf13/cobra"

	"github.com/submariner-io/shipyard/test/e2e"
	_ "github.com/submariner-io/submariner/test/e2e/dataplane"
	_ "github.com/submariner-io/submariner/test/e2e/framework"
)

func init() {
	addVerifyFlags(verifyConnectivityCmd)
	rootCmd.AddCommand(verifyConnectivityCmd)
}

var verifyConnectivityCmd = &cobra.Command{
	Deprecated: "Use verify --connectivity",
	Use:        "verify-connectivity <kubeConfig1> <kubeConfig2>",
	Short:      "Verify connectivity between two clusters",
	Args: func(cmd *cobra.Command, args []string) error {
		return checkValidateArguments(args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		configureTestingFramework(args)
		e2e.RunE2ETests(&testing.T{})
	},
}
