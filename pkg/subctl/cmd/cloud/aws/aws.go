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
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/submariner-io/admiral/pkg/util"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	aws "github.com/submariner-io/cloud-prepare/pkg/aws"
	"github.com/submariner-io/cloud-prepare/pkg/ocp"
	"github.com/submariner-io/submariner-operator/internal/exit"
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
	command.Flags().StringVar(&profile, "profile", aws.DefaultProfile(), "AWS profile to use for credentials")
	command.Flags().StringVar(&credentialsFile, "credentials", aws.DefaultCredentialsFile(), "AWS credentials configuration file")
}

// RunOnAWS runs the given function on AWS, supplying it with a cloud instance connected to AWS and a reporter that writes to CLI.
// The functions makes sure that infraID and region are specified, and extracts the credentials from a secret in order to connect to AWS.
func RunOnAWS(restConfigProducer restconfig.Producer, gwInstanceType string,
	function func(cloud api.Cloud, gwDeployer api.GatewayDeployer, reporter api.Reporter) error) error {
	if ocpMetadataFile != "" {
		err := initializeFlagsFromOCPMetadata(ocpMetadataFile)
		exit.OnErrorWithMessage(err, "Failed to read AWS information from OCP metadata file")
	} else {
		utils.ExpectFlag(infraIDFlag, infraID)
		utils.ExpectFlag(regionFlag, region)
	}

	reporter := cloudutils.NewStatusReporter()

	reporter.Started("Initializing AWS connectivity")

	awsCloud, err := aws.NewCloudFromSettings(credentialsFile, profile, infraID, region)
	if err != nil {
		reporter.Failed(err)

		return errors.Wrap(err, "error loading default config")
	}

	reporter.Succeeded("")

	k8sConfig, err := restConfigProducer.ForCluster()
	exit.OnErrorWithMessage(err, "Failed to initialize a Kubernetes config")

	restMapper, err := util.BuildRestMapper(k8sConfig)
	exit.OnErrorWithMessage(err, "Failed to create restmapper")

	dynamicClient, err := dynamic.NewForConfig(k8sConfig)
	exit.OnErrorWithMessage(err, "Failed to create dynamic client")

	msDeployer := ocp.NewK8sMachinesetDeployer(restMapper, dynamicClient)
	gwDeployer, err := aws.NewOcpGatewayDeployer(awsCloud, msDeployer, gwInstanceType)
	exit.OnErrorWithMessage(err, "Failed to initialize a GatewayDeployer config")

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
