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

// Package rhos provides common functionality to run cloud prepare/cleanup on RHOS Clusters.
package rhos

import (
	"github.com/spf13/cobra"
)

const (
	infraIDFlag    = "infra-id"
	regionFlag     = "region"
	projectIDFlag  = "project-id"
	cloudEntryFlag = "cloud-entry"
)

var (
	infraID         string
	region          string
	projectID       string
	ocpMetadataFile string
	cloudEntry      string
)

// AddRHOSFlags adds basic flags needed by RHOS.
func AddRHOSFlags(command *cobra.Command) {
	command.Flags().StringVar(&infraID, infraIDFlag, "", "RHOS infra ID")
	command.Flags().StringVar(&region, regionFlag, "", "RHOS region")
	command.Flags().StringVar(&projectID, projectIDFlag, "", "RHOS project ID")
	command.Flags().StringVar(&ocpMetadataFile, "ocp-metadata", "",
		"OCP metadata.json file (or the directory containing it) from which to read the RHOS infra ID "+
			"and region from (takes precedence over the specific flags)")
	command.Flags().StringVar(&cloudEntry, cloudEntryFlag, "", "the cloud entry to use")
}
