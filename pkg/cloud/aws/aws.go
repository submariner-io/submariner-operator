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
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/admiral/pkg/util"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/cloud-prepare/pkg/aws"
	"github.com/submariner-io/cloud-prepare/pkg/ocp"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"k8s.io/client-go/dynamic"
)

type Config struct {
	Gateways        int
	InfraID         string
	Region          string
	Profile         string
	CredentialsFile string
	OcpMetadataFile string
	GWInstanceType  string
}

// RunOn runs the given function on AWS, supplying it with a cloud instance connected to AWS and a reporter that writes to CLI.
// The functions makes sure that infraID and region are specified, and extracts the credentials from a secret in order to connect to AWS.
func RunOn(restConfigProducer restconfig.Producer, config *Config, status reporter.Interface,
	function func(api.Cloud, api.GatewayDeployer, reporter.Interface) error,
) error {
	status.Start("Initializing AWS connectivity")

	awsCloud, err := aws.NewCloudFromSettings(config.CredentialsFile, config.Profile, config.InfraID, config.Region)
	if err != nil {
		return status.Error(err, "error loading default config")
	}

	status.End()

	k8sConfig, err := restConfigProducer.ForCluster()
	if err != nil {
		return status.Error(err, "error initializing Kubernetes config")
	}

	restMapper, err := util.BuildRestMapper(k8sConfig.Config)
	if err != nil {
		return status.Error(err, "error creating REST mapper")
	}

	dynamicClient, err := dynamic.NewForConfig(k8sConfig.Config)
	if err != nil {
		return status.Error(err, "error creating dynamic client")
	}

	msDeployer := ocp.NewK8sMachinesetDeployer(restMapper, dynamicClient)

	gwDeployer, err := aws.NewOcpGatewayDeployer(awsCloud, msDeployer, config.GWInstanceType)
	if err != nil {
		return status.Error(err, "error creating the gateway deployer")
	}

	return function(awsCloud, gwDeployer, status)
}

func ReadFromFile(metadataFile string) (string, string, error) {
	fileInfo, err := os.Stat(metadataFile)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to stat file %q", metadataFile)
	}

	if fileInfo.IsDir() {
		metadataFile = filepath.Join(metadataFile, "metadata.json")
	}

	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return "", "", errors.Wrapf(err, "error reading file %q", metadataFile)
	}

	var metadata struct {
		InfraID string `json:"infraID"`
		AWS     struct {
			Region string `json:"region"`
		} `json:"aws"`
	}

	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return "", "", errors.Wrap(err, "error unmarshalling data")
	}

	return metadata.InfraID, metadata.AWS.Region, nil
}
