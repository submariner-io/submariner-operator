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
	"github.com/submariner-io/submariner-operator/pkg/cloud/cleanup"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/gcp"
)

// newGCPCleanupCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func newGCPCleanupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gcp",
		Short: "Clean up a GCP cloud",
		Long:  "This command cleans up an installer-provisioned infrastructure (IPI) on GCP-based cloud after Submariner uninstallation.",
		Run:   cleanup.GCP,
	}

	gcp.AddGCPFlags(cmd)

	return cmd
}
