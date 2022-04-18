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

// Package aws provides common functionality to run cloud prepare/cleanup on AWS.
package aws

import (
	"github.com/spf13/cobra"
	cpaws "github.com/submariner-io/cloud-prepare/pkg/aws"
	"github.com/submariner-io/submariner-operator/pkg/cloud/aws"
)

const (
	infraIDFlag = "infra-id"
	regionFlag  = "region"
)

// AddAWSFlags adds basic flags needed by AWS.
func AddAWSFlags(command *cobra.Command, config *aws.Config) {
	command.Flags().StringVar(&config.InfraID, infraIDFlag, "", "AWS infra ID")
	command.Flags().StringVar(&config.Region, regionFlag, "", "AWS region")
	command.Flags().StringVar(&config.OcpMetadataFile, "ocp-metadata", "",
		"OCP metadata.json file (or directory containing it) to read AWS infra ID and region from (Takes precedence over the flags)")
	command.Flags().StringVar(&config.Profile, "profile", cpaws.DefaultProfile(), "AWS profile to use for credentials")
	command.Flags().StringVar(&config.CredentialsFile, "credentials", cpaws.DefaultCredentialsFile(), "AWS credentials configuration file")
}
