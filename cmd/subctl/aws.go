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
	cpaws "github.com/submariner-io/cloud-prepare/pkg/aws"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/exit"
	cloudaws "github.com/submariner-io/submariner-operator/pkg/cloud/aws"
	"github.com/submariner-io/submariner-operator/pkg/cloud/cleanup"
	"github.com/submariner-io/submariner-operator/pkg/cloud/prepare"
)

var (
	awsConfig cloudaws.Config

	awsPrepareCmd = &cobra.Command{
		Use:     "aws",
		Short:   "Prepare an OpenShift AWS cloud",
		Long:    "This command prepares an OpenShift installer-provisioned infrastructure (IPI) on AWS cloud for Submariner installation.",
		PreRunE: checkAWSFlags,
		Run: func(cmd *cobra.Command, args []string) {
			err := prepare.AWS(&restConfigProducer, &cloudPorts, &awsConfig, cli.NewReporter())
			exit.OnError(err)
		},
	}

	awsCleanupCmd = &cobra.Command{
		Use:   "aws",
		Short: "Clean up an AWS cloud",
		Long: "This command cleans up an OpenShift installer-provisioned infrastructure (IPI) on AWS-based" +
			" cloud after Submariner uninstallation.",
		PreRunE: checkAWSFlags,
		Run: func(cmd *cobra.Command, args []string) {
			err := cleanup.AWS(&restConfigProducer, &awsConfig, cli.NewReporter())
			exit.OnError(err)
		},
	}
)

func init() {
	addGeneralAWSFlags := func(command *cobra.Command) {
		command.Flags().StringVar(&awsConfig.InfraID, infraIDFlag, "", "AWS infra ID")
		command.Flags().StringVar(&awsConfig.Region, regionFlag, "", "AWS region")
		command.Flags().StringVar(&awsConfig.OcpMetadataFile, "ocp-metadata", "",
			"OCP metadata.json file (or directory containing it) to read AWS infra ID and region from (Takes precedence over the flags)")
		command.Flags().StringVar(&awsConfig.Profile, "profile", cpaws.DefaultProfile(), "AWS profile to use for credentials")
		command.Flags().StringVar(&awsConfig.CredentialsFile, "credentials", cpaws.DefaultCredentialsFile(), "AWS credentials configuration file")
	}

	addGeneralAWSFlags(awsPrepareCmd)
	awsPrepareCmd.Flags().StringVar(&awsConfig.GWInstanceType, "gateway-instance", "c5d.large", "Type of gateways instance machine")
	awsPrepareCmd.Flags().IntVar(&awsConfig.Gateways, "gateways", defaultNumGateways,
		"Number of dedicated gateways to deploy (Set to `0` when using --load-balancer mode)")

	cloudPrepareCmd.AddCommand(awsPrepareCmd)

	addGeneralAWSFlags(awsCleanupCmd)
	cloudCleanupCmd.AddCommand(awsCleanupCmd)
}

func checkAWSFlags(cmd *cobra.Command, args []string) error {
	if awsConfig.OcpMetadataFile == "" {
		expectFlag(infraIDFlag, awsConfig.InfraID)
		expectFlag(regionFlag, awsConfig.Region)
	}

	return nil
}
