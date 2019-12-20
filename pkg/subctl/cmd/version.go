/*
© 2019 Red Hat, Inc. and others.

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

	"github.com/spf13/cobra"

	"github.com/submariner-io/submariner-operator/pkg/version"
)

// infoCmd represents the info command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Get version information on subctl",
	Long: `This command shows the version tag, and git commit for your
subctl binary.`,
	Run: subctlVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func subctlVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("subctl version: %s\n", version.Version)
	fmt.Printf("built from git commit id: %s\n", version.Commit)
}
