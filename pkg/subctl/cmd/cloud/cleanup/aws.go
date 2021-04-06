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

package cleanup

import (
	"github.com/spf13/cobra"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	cloudutils "github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/utils"

	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
)

var (
	infraID string
	region  string
)

// NewCommand returns a new cobra.Command used to prepare a cloud infrastructure
func newAWSCleanupCommand() *cobra.Command {
	awsCloudPrepareCmd := &cobra.Command{
		Use:   "aws",
		Short: "Clean up an AWS cloud",
		Long:  "This command cleans up an AWS based cloud after Submariner uninstallation.",
		Run:   cleanupAws,
	}

	awsCloudPrepareCmd.Flags().StringVar(&infraID, "infra-id", "", "AWS infra ID")
	awsCloudPrepareCmd.Flags().StringVar(&region, "region", "", "AWS region")

	return awsCloudPrepareCmd
}

func cleanupAws(cmd *cobra.Command, args []string) {
	err := cloudutils.RunOnAWS(infraID, region, "", *kubeConfig, *kubeContext,
		func(cloud api.Cloud, reporter api.Reporter) error {
			return cloud.CleanupAfterSubmariner(reporter)
		})

	utils.ExitOnError("Failed to cleanup AWS cloud", err)
}
