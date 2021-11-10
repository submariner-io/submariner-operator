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

// This package provides common functionality to run cloud prepare/cleanup on AWS
package aws

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/spf13/cobra"
	"github.com/submariner-io/admiral/pkg/util"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	cloudprepareaws "github.com/submariner-io/cloud-prepare/pkg/aws"
	"github.com/submariner-io/cloud-prepare/pkg/ocp"
	cloudutils "github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils/restconfig"
	"gopkg.in/ini.v1"
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

// AddAWSFlags adds basic flags needed by AWS
func AddAWSFlags(command *cobra.Command) {
	command.Flags().StringVar(&infraID, infraIDFlag, "", "AWS infra ID")
	command.Flags().StringVar(&region, regionFlag, "", "AWS region")
	command.Flags().StringVar(&ocpMetadataFile, "ocp-metadata", "",
		"OCP metadata.json file (or directory containing it) to read AWS infra ID and region from (Takes precedence over the flags)")
	command.Flags().StringVar(&profile, "profile", "default", "AWS profile to use for credentials")

	dirname, err := os.UserHomeDir()
	if err != nil {
		utils.ExitOnError("failed to find home directory", err)
	}

	defaultCredentials := filepath.FromSlash(fmt.Sprintf("%s/.aws/credentials", dirname))
	command.Flags().StringVar(&credentialsFile, "credentials", defaultCredentials, "AWS credentials configuration file")
}

// RunOnAWS runs the given function on AWS, supplying it with a cloud instance connected to AWS and a reporter that writes to CLI.
// The functions makes sure that infraID and region are specified, and extracts the credentials from a secret in order to connect to AWS.
func RunOnAWS(gwInstanceType, kubeConfig, kubeContext string,
	function func(cloud api.Cloud, gwDeployer api.GatewayDeployer, reporter api.Reporter) error) error {
	if ocpMetadataFile != "" {
		err := initializeFlagsFromOCPMetadata(ocpMetadataFile)
		utils.ExitOnError("Failed to read AWS information from OCP metadata file", err)
	} else {
		utils.ExpectFlag(infraIDFlag, infraID)
		utils.ExpectFlag(regionFlag, region)
	}

	reporter := cloudutils.NewCLIReporter()
	reporter.Started("Retrieving AWS credentials from your AWS configuration")
	creds, err := getAWSCredentials()
	if err != nil {
		reporter.Failed(err)
		return err
	}
	reporter.Succeeded("")

	reporter.Started("Initializing AWS connectivity")
	awsConfig := aws.Config{
		Credentials: creds,
		Region:      aws.String(region),
	}

	awsSession, err := session.NewSession(&awsConfig)
	if err != nil {
		reporter.Failed(err)
		return err
	}
	reporter.Succeeded("")

	k8sConfig, err := restconfig.ForCluster(kubeConfig, kubeContext)
	utils.ExitOnError("Failed to initialize a Kubernetes config", err)

	restMapper, err := util.BuildRestMapper(k8sConfig)
	utils.ExitOnError("Failed to create restmapper", err)

	dynamicClient, err := dynamic.NewForConfig(k8sConfig)
	utils.ExitOnError("Failed to create dynamic client", err)

	awsCloud := cloudprepareaws.NewCloud(ec2.New(awsSession), infraID, region)
	msDeployer := ocp.NewK8sMachinesetDeployer(restMapper, dynamicClient)
	gwDeployer, err := cloudprepareaws.NewOcpGatewayDeployer(awsCloud, msDeployer, gwInstanceType)
	utils.ExitOnError("Failed to initialize a GatewayDeployer config", err)

	return function(awsCloud, gwDeployer, reporter)
}

func initializeFlagsFromOCPMetadata(metadataFile string) error {
	fileInfo, err := os.Stat(metadataFile)
	if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		metadataFile = filepath.Join(metadataFile, "metadata.json")
	}

	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return err
	}

	var metadata struct {
		InfraID string `json:"infraID"`
		AWS     struct {
			Region string `json:"region"`
		} `json:"aws"`
	}

	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return err
	}

	infraID = metadata.InfraID
	region = metadata.AWS.Region
	return nil
}

// Retrieve AWS credentials from the AWS credentials file.
func getAWSCredentials() (*credentials.Credentials, error) {
	cfg, err := ini.Load(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read AWS credentials from %s: %w", credentialsFile, err)
	}

	profileSection, err := cfg.GetSection(profile)
	if err != nil {
		return nil, fmt.Errorf("failed to find profile %s in AWS credentials file %s", profile, credentialsFile)
	}

	accessKeyID, err := profileSection.GetKey("aws_access_key_id")
	if err != nil {
		return nil, fmt.Errorf("failed to find access key ID in profile %s in AWS credentials file %s", profile, credentialsFile)
	}

	secretAccessKey, err := profileSection.GetKey("aws_secret_access_key")
	if err != nil {
		return nil, fmt.Errorf("failed to find secret access key in profile %s in AWS credentials file %s", profile, credentialsFile)
	}

	return credentials.NewStaticCredentials(accessKeyID.String(), secretAccessKey.String(), ""), nil
}
