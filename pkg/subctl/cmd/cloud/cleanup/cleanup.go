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

package cleanup

import (
	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
)

var parentRestConfigProducer *restconfig.Producer

// NewCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func NewCommand(restConfigProducer *restconfig.Producer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up the cloud",
		Long:  `This command cleans up the cloud after Submariner uninstallation.`,
	}
	parentRestConfigProducer = restConfigProducer

	cmd.AddCommand(newAWSCleanupCommand())
	cmd.AddCommand(newGCPCleanupCommand())
	cmd.AddCommand(newGenericCleanupCommand())

	return cmd
}
