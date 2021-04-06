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

package cloud

import (
	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/cleanup"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/prepare"
)

// NewCommand returns a new cobra.Command used to prepare a cloud infrastructure
func NewCommand(origKubeConfig, origKubeContext *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud",
		Short: "Cloud operations",
		Long:  `This command contains cloud operations relating to Submariner installation.`,
	}

	cmd.AddCommand(prepare.NewCommand(origKubeConfig, origKubeContext))
	cmd.AddCommand(cleanup.NewCommand(origKubeConfig, origKubeContext))

	return cmd
}
