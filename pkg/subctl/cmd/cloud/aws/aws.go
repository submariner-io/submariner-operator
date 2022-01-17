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

// This package provides common functionality to run cloud prepare/cleanup on AWS.
package aws

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/submariner-io/admiral/pkg/util"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	cloudprepareaws "github.com/submariner-io/cloud-prepare/pkg/aws"
	"github.com/submariner-io/cloud-prepare/pkg/ocp"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	cloudutils "github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"k8s.io/client-go/dynamic"
)

const (
	infraIDFlag = "infra-id"
	regionFlag  = "region"
)

var (
	infraID         string
	region          string
	profile         string
	credentialsFile string
	ocpMetadataFile string
)

// AddAWSFlags adds basic flags needed by AWS.
func AddAWSFlags(command *cobra.Command) {
	command.Flags().StringVar(&infraID, infraIDFlag, "", "AWS infra ID")
	command.Flags().StringVar(&region, regionFlag, "", "AWS region")
	command.Flags().StringVar(&ocpMetadataFile, "ocp-metadata", "",
		"OCP metadata.json file (or directory containing it) to read AWS infra ID and region from (Takes precedence over the flags)")
	command.Flags().StringVar(&profile, "profile", "default", "AWS profile to use for credentials")
	command.Flags().StringVar(&credentialsFile, "credentials", config.DefaultSharedCredentialsFilename(), "AWS credentials configuration file")
}

// RunOnAWS runs the given function on AWS, supplying it with a cloud instance connected to AWS and a reporter that writes to CLI.
// The functions makes sure that infraID and region are specified, and extracts the credentials from a secret in order to connect to AWS.
func RunOnAWS(restConfigProducer restconfig.Producer, gwInstanceType string,
	function func(cloud api.Cloud, gwDeployer api.GatewayDeployer, reporter api.Reporter) error) error {
	if ocpMetadataFile != "" {
		err := initializeFlagsFromOCPMetadata(ocpMetadataFile)
		utils.ExitOnError("Failed to read AWS information from OCP metadata file", err)
	} else {
		utils.ExpectFlag(infraIDFlag, infraID)
		utils.ExpectFlag(regionFlag, region)
	}

	reporter := cloudutils.NewStatusReporter()

	reporter.Started("Initializing AWS connectivity")

	options := []func(*config.LoadOptions) error{config.WithRegion(region), config.WithSharedConfigProfile(profile)}
	if credentialsFile != config.DefaultSharedCredentialsFilename() {
		options = append(options, config.WithSharedCredentialsFiles([]string{credentialsFile}))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), options...)
	if err != nil {
		reporter.Failed(err)

		return errors.Wrap(err, "error loading default config")
	}

	ec2Client := ec2.NewFromConfig(cfg)

	reporter.Succeeded("")

	k8sConfig, err := restConfigProducer.ForCluster()
	utils.ExitOnError("Failed to initialize a Kubernetes config", err)

	restMapper, err := util.BuildRestMapper(k8sConfig)
	utils.ExitOnError("Failed to create restmapper", err)

	dynamicClient, err := dynamic.NewForConfig(k8sConfig)
	utils.ExitOnError("Failed to create dynamic client", err)

	awsCloud := cloudprepareaws.NewCloud(ec2Client, infraID, region)
	msDeployer := ocp.NewK8sMachinesetDeployer(restMapper, dynamicClient)
	gwDeployer, err := cloudprepareaws.NewOcpGatewayDeployer(awsCloud, msDeployer, gwInstanceType)
	utils.ExitOnError("Failed to initialize a GatewayDeployer config", err)

	return function(awsCloud, gwDeployer, reporter)
}

func initializeFlagsFromOCPMetadata(metadataFile string) error {
	fileInfo, err := os.Stat(metadataFile)
	if err != nil {
		return errors.Wrapf(err, "failed to stat file %q", metadataFile)
	}

	if fileInfo.IsDir() {
		metadataFile = filepath.Join(metadataFile, "metadata.json")
	}

	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return errors.Wrapf(err, "error reading file %q", metadataFile)
	}

	var metadata struct {
		InfraID string `json:"infraID"`
		AWS     struct {
			Region string `json:"region"`
		} `json:"aws"`
	}

	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling data")
	}

	infraID = metadata.InfraID
	region = metadata.AWS.Region

	return nil
}
