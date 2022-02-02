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
	"fmt"
	"github.com/spf13/cobra"
	"github.com/submariner-io/admiral/pkg/util"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/cloud-prepare/pkg/aws"
	"github.com/submariner-io/cloud-prepare/pkg/k8s"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/cloud"
	cloudaws "github.com/submariner-io/submariner-operator/pkg/cloud/aws"
	cloudgcp "github.com/submariner-io/submariner-operator/pkg/cloud/gcp"
	"github.com/submariner-io/submariner-operator/pkg/cloud/prepare"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"os"
	"path/filepath"
)

var (
	cloudCmd = &cobra.Command{
		Use:   "cloud",
		Short: "Cloud operations",
		Long:  `This command contains cloud operations related to Submariner installation.`,
	}
	port       cloud.Port
	awsOptions cloudaws.Options
	gcpOptions cloudgcp.Options
)

const (
	DefaultNumGateways = 1
    InfraID        = "infra-id"
    Region         = "region"
	ProjectID = "project-id"
)

func init() {
	restConfigProducer.AddKubeContextFlag(cloudCmd)
	k8sConfig, err := restConfigProducer.ForCluster()
	exit.OnErrorWithMessage(err, "Failed to initialize a Kubernetes config")

	reporter := reporter.NewCloudReporter()

	cloudCmd.AddCommand(newPrepareCommand(k8sConfig, reporter))
	rootCmd.AddCommand(cloudCmd)
}

// addAWSFlags adds basic flags needed by AWS.
func addAWSFlags(command *cobra.Command) {
	command.Flags().StringVar(&awsOptions.InfraID, InfraID, "", "AWS infra ID")
	command.Flags().StringVar(&awsOptions.Region, Region, "", "AWS region")
	command.Flags().StringVar(&awsOptions.OcpMetadataFile, "ocp-metadata", "",
		"OCP metadata.json file (or directory containing it) to read AWS infra ID and region from (Takes precedence over the flags)")
	command.Flags().StringVar(&awsOptions.Profile, "profile", aws.DefaultProfile(), "AWS profile to use for credentials")
	command.Flags().StringVar(&awsOptions.CredentialsFile, "credentials", aws.DefaultCredentialsFile(), "AWS credentials configuration file")
}

// newPrepareCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func newPrepareCommand(config *rest.Config, reporter api.Reporter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare the cloud",
		Long:  `This command prepares the cloud for Submariner installation.`,
	}

	cmd.PersistentFlags().Uint16Var(&port.Natt, "natt-port", 4500, "IPSec NAT traversal port")
	cmd.PersistentFlags().Uint16Var(&port.NatDiscovery, "nat-discovery-port", 4490, "NAT discovery port")
	cmd.PersistentFlags().Uint16Var(&port.VxLAN, "vxlan-port", 4800, "Internal VxLAN port")
	cmd.PersistentFlags().Uint16Var(&port.Metrics, "metrics-port", 8080, "Metrics port")

	cmd.AddCommand(newAWSPrepareCommand(config, reporter))
	cmd.AddCommand(newGCPPrepareCommand(config, reporter))

	return cmd
}

// NewCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func newAWSPrepareCommand(config *rest.Config, reporter api.Reporter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aws",
		Short: "Prepare an OpenShift AWS cloud",
		Long:  "This command prepares an OpenShift installer-provisioned infrastructure (IPI) on AWS cloud for Submariner installation.",
		Run: func(cmd *cobra.Command, args []string) {
			restMapper, err := util.BuildRestMapper(config)
			exit.OnErrorWithMessage(err, "Failed to create restmapper")

			dynamicClient, err := dynamic.NewForConfig(config)
			exit.OnErrorWithMessage(err, "Failed to create dynamic client")

			if (awsOptions.InfraID == "" || awsOptions.Region == "") && awsOptions.OcpMetadataFile == "" {
				exit.WithMessage("You must specify the infra-ID and region flags or OCP metadata.json file via ocp-metadata flag")
			}

			// Values from metadata file takes precedence over the values provided via flags
			if awsOptions.OcpMetadataFile != "" {
				awsOptions.InfraID, awsOptions.Region, err = cloudaws.ReadFromFile(awsOptions.OcpMetadataFile)
				exit.OnErrorWithMessage(err, "Failed to read AWS information from OCP metadata file")
			}

			err = prepare.AWS(port, &awsOptions, restMapper, dynamicClient, reporter)
			exit.OnErrorWithMessage(err, "Failed to prepare AWS cloud")
		},
	}

	addAWSFlags(cmd)
	cmd.Flags().StringVar(&awsOptions.GWInstance, "gateway-instance", "c5d.large", "Type of gateways instance machine")
	cmd.Flags().IntVar(&awsOptions.Gateways, "gateways", DefaultNumGateways,
		"Number of dedicated gateways to deploy (Set to `0` when using --load-balancer mode)")

	return cmd
}

// NewCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func newGCPPrepareCommand(config *rest.Config, reporter api.Reporter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gcp",
		Short: "Prepare an OpenShift GCP cloud",
		Long:  "This command prepares an OpenShift installer-provisioned infrastructure (IPI) on GCP cloud for Submariner installation.",
		Run:   func(cmd *cobra.Command, args []string) {
			restMapper, err := util.BuildRestMapper(config)
			exit.OnErrorWithMessage(err, "Failed to create restmapper")

			dynamicClient, clientSet, err := restconfig.Clients(config)
			exit.OnErrorWithMessage(err, "Failed to create client")

			k8sClientSet := k8s.NewInterface(clientSet)

			if (gcpOptions.InfraID == "" || gcpOptions.Region == "" || gcpOptions.ProjectID == "") && gcpOptions.OcpMetadataFile == "" {
				exit.WithMessage("You must specify the infra-ID, region and Project ID flags or OCP metadata.json file via ocp-metadata flag")
			}

			// Values from metadata file takes precedence over the values provided via flags
			if gcpOptions.OcpMetadataFile != "" {
				gcpOptions.InfraID, gcpOptions.Region, gcpOptions.ProjectID, err = cloudgcp.ReadFromFile(gcpOptions.OcpMetadataFile)
				exit.OnErrorWithMessage(err, "Failed to read AWS information from OCP metadata file")
			}

			reporter.Started("Retrieving GCP credentials from your GCP configuration")

			creds, err := cloudgcp.GetCredentials(gcpOptions.CredentialsFile)
			exit.OnErrorWithMessage(err, "Failed to get GCP credentials")
			reporter.Succeeded("")

			err = prepare.GCP(port, &gcpOptions, creds, restMapper, dynamicClient, k8sClientSet, reporter)
			exit.OnErrorWithMessage(err, "Failed to prepare GCP cloud")
		},
	}

	addGCPFlags(cmd)
	cmd.Flags().StringVar(&gcpOptions.GWInstance, "gateway-instance", "n1-standard-4", "Type of gateway instance machine")
	cmd.Flags().IntVar(&gcpOptions.Gateways, "gateways", DefaultNumGateways,
		"Number of gateways to deploy")
	cmd.Flags().BoolVar(&gcpOptions.DedicatedGW, "dedicated-gateway", false,
		"Whether a dedicated gateway node has to be deployed (default false)")

	return cmd
}

// addGCPFlags adds basic flags needed by GCP.
func addGCPFlags(command *cobra.Command) {
	command.Flags().StringVar(&gcpOptions.InfraID, InfraID, "", "GCP infra ID")
	command.Flags().StringVar(&gcpOptions.Region, Region, "", "GCP region")
	command.Flags().StringVar(&gcpOptions.ProjectID, ProjectID, "", "GCP project ID")
	command.Flags().StringVar(&gcpOptions.OcpMetadataFile, "ocp-metadata", "",
		"OCP metadata.json file (or the directory containing it) from which to read the GCP infra ID "+
			"and region from (takes precedence over the specific flags)")

	dirname, err := os.UserHomeDir()
	if err != nil {
		exit.OnErrorWithMessage(err, "failed to find home directory")
	}

	defaultCredentials := filepath.FromSlash(fmt.Sprintf("%s/.gcp/osServiceAccount.json", dirname))
	command.Flags().StringVar(&gcpOptions.CredentialsFile, "credentials", defaultCredentials, "GCP credentials configuration file")
}