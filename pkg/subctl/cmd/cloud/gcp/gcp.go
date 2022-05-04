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

// Package gcp provides common functionality to run cloud prepare/cleanup on GCP Clusters.
package gcp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/exit"
	cloudgcp "github.com/submariner-io/submariner-operator/pkg/cloud/gcp"
)

const (
	infraIDFlag   = "infra-id"
	regionFlag    = "region"
	projectIDFlag = "project-id"
)

// AddGCPFlags adds basic flags needed by GCP.
func AddGCPFlags(command *cobra.Command, config *cloudgcp.Config) {
	command.Flags().StringVar(&config.InfraID, infraIDFlag, "", "GCP infra ID")
	command.Flags().StringVar(&config.Region, regionFlag, "", "GCP region")
	command.Flags().StringVar(&config.ProjectID, projectIDFlag, "", "GCP project ID")
	command.Flags().StringVar(&config.OcpMetadataFile, "ocp-metadata", "",
		"OCP metadata.json file (or the directory containing it) from which to read the GCP infra ID "+
			"and region from (takes precedence over the specific flags)")

	dirname, err := os.UserHomeDir()
	if err != nil {
		exit.OnErrorWithMessage(err, "failed to find home directory")
	}

	defaultCredentials := filepath.FromSlash(fmt.Sprintf("%s/.gcp/osServiceAccount.json", dirname))
	command.Flags().StringVar(&config.CredentialsFile, "credentials", defaultCredentials, "GCP credentials configuration file")
}
