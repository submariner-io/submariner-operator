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
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/submariner-operator/cmd/subctl/cloud/cleanup"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/gcp"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
)

func GCP(cmd *cobra.Command, args []string) {
	err := gcp.RunOnGCP(*cleanup.parentRestConfigProducer, "", false,
		// nolint:wrapcheck // No need to wrap errors here
		func(cloud api.Cloud, gwDeployer api.GatewayDeployer, reporter api.Reporter) error {
			err := gwDeployer.Cleanup(reporter)
			if err != nil {
				return err
			}

			return cloud.CleanupAfterSubmariner(reporter)
		})

	utils.ExitOnError("Failed to cleanup GCP cloud", err)
}
