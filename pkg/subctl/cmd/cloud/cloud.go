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

package cloud

import (
	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/cleanup"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/prepare"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/vpcpeering"
)

var (
	cloudCmd = &cobra.Command{
		Use:   "cloud",
		Short: "Cloud operations",
		Long:  `This command contains cloud operations related to Submariner installation.`,
	}
	restConfigProducer = restconfig.NewProducer()
)

func init() {
	cloudCmd.AddCommand(prepare.NewCommand(&restConfigProducer))
	cloudCmd.AddCommand(cleanup.NewCommand(&restConfigProducer))
	cloudCmd.AddCommand(vpcpeering.NewCommand(&restConfigProducer))
	restConfigProducer.AddKubeContextFlag(cloudCmd)
	cmd.AddToRootCommand(cloudCmd)
}
