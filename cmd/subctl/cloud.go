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

package subctl

import (
	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/cloud"
)

const (
	infraIDFlag        = "infra-id"
	regionFlag         = "region"
	defaultNumGateways = 1
	projectIDFlag      = "project-id"
	cloudEntryFlag     = "cloud-entry"
)

var (
	cloudPorts cloud.Ports

	cloudCmd = &cobra.Command{
		Use:   "cloud",
		Short: "Cloud operations",
		Long:  `This command contains cloud operations related to Submariner installation.`,
	}

	cloudCleanupCmd = &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up the cloud",
		Long:  `This command cleans up the cloud after Submariner uninstallation.`,
	}

	cloudPrepareCmd = &cobra.Command{
		Use:   "prepare",
		Short: "Prepare the cloud",
		Long:  `This command prepares the cloud for Submariner installation.`,
	}
)

func init() {
	restConfigProducer.AddKubeContextFlag(cloudCmd)
	rootCmd.AddCommand(cloudCmd)

	cloudPrepareCmd.PersistentFlags().Uint16Var(&cloudPorts.Natt, "natt-port", 4500, "IPSec NAT traversal port")
	cloudPrepareCmd.PersistentFlags().Uint16Var(&cloudPorts.NatDiscovery, "nat-discovery-port", 4490, "NAT discovery port")
	cloudPrepareCmd.PersistentFlags().Uint16Var(&cloudPorts.Vxlan, "vxlan-port", 4800, "Internal VXLAN port")
	cloudPrepareCmd.PersistentFlags().Uint16Var(&cloudPorts.Metrics, "metrics-port", 8080, "Metrics port")

	cloudCmd.AddCommand(cloudPrepareCmd)

	cloudCmd.AddCommand(cloudCleanupCmd)
}
