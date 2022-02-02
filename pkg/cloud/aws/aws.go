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
	"k8s.io/apimachinery/pkg/api/meta"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	aws "github.com/submariner-io/cloud-prepare/pkg/aws"
	"github.com/submariner-io/cloud-prepare/pkg/ocp"
	"k8s.io/client-go/dynamic"
)

type Options struct {
	InfraID         string
	Region          string
	Profile         string
	CredentialsFile string
	OcpMetadataFile string
	GWInstance      string
	Gateways        int
}

// RunOnAWS runs the given function on AWS, supplying it with a cloud instance connected to AWS and a reporter that writes to CLI.
// The functions makes sure that infraID and region are specified, and extracts the credentials from a secret in order to connect to AWS.
func RunOnAWS(info *Options, restMapper meta.RESTMapper, dynamicClient dynamic.Interface, gwInstanceType string,
	function func(cloud api.Cloud, gwDeployer api.GatewayDeployer, reporter api.Reporter) error, reporter api.Reporter) error {

	reporter.Started("Initializing AWS connectivity")

	awsCloud, err := aws.NewCloudFromSettings(info.CredentialsFile, info.Profile, info.InfraID, info.Region)
	if err != nil {
		reporter.Failed(err)

		return errors.Wrap(err, "error loading default config")
	}

	reporter.Succeeded("")

	msDeployer := ocp.NewK8sMachinesetDeployer(restMapper, dynamicClient)

	gwDeployer, err := aws.NewOcpGatewayDeployer(awsCloud, msDeployer, gwInstanceType)
	if err != nil {
		return errors.Wrapf(err, "Failed to initialize a GatewayDeployer config")
	}

	return function(awsCloud, gwDeployer, reporter)
}

func ReadFromFile(filename string) (string, string, error) {
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to stat file %q", filename)
	}

	metadataFile := filename

	if fileInfo.IsDir() {
		metadataFile = filepath.Join(filename, "metadata.json")
	}

	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return "", "",errors.Wrapf(err, "error reading file %q", metadataFile)
	}

	var metadata struct {
		InfraID string `json:"infraID"`
		AWS     struct {
			Region string `json:"region"`
		} `json:"aws"`
	}

	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return "", "",errors.Wrap(err, "error unmarshalling data")
	}

	return metadata.InfraID, metadata.AWS.Region, nil

}
