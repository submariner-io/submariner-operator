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

package diagnose

import (
	"github.com/spf13/cobra"
)

var diagnoseFirewallConfigCmd = &cobra.Command{
	Use:   "firewall",
	Short: "Check the firewall configuration",
	Long:  "This command checks if the firewall is configured as per Submariner pre-requisites.",
}

var validationTimeout uint

func addDiagnoseFWConfigFlags(command *cobra.Command) {
	command.Flags().UintVar(&validationTimeout, "validation-timeout", 90,
		"timeout in seconds while validating the connection attempt")
	addNamespaceFlag(command)
}

func init() {
	diagnoseCmd.AddCommand(diagnoseFirewallConfigCmd)
}