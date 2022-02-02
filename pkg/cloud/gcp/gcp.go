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
	"context"
	"encoding/json"
	"github.com/submariner-io/releases/projects/cloud-prepare/pkg/k8s"
	"k8s.io/apimachinery/pkg/api/meta"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/cloud-prepare/pkg/gcp"
	gcpClientIface "github.com/submariner-io/cloud-prepare/pkg/gcp/client"
	"github.com/submariner-io/cloud-prepare/pkg/ocp"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/option"
	"k8s.io/client-go/dynamic"
)

type Options struct {
	InfraID         string
	Region          string
	ProjectID       string
	CredentialsFile string
	OcpMetadataFile string
	GWInstance      string
	Gateways        int
	DedicatedGW     bool
}


// RunOnGCP runs the given function on GCP, supplying it with a cloud instance connected to GCP and a reporter that writes to CLI.
// The functions makes sure that infraID and region are specified, and extracts the credentials from a secret in order to connect to GCP.
func RunOnGCP(info *Options, creds *google.Credentials, restMapper meta.RESTMapper, dynamicClient dynamic.Interface, k8sClientSet k8s.Interface, gwInstanceType string,
	function func(cloud api.Cloud, gwDeployer api.GatewayDeployer, reporter api.Reporter) error, reporter api.Reporter) error {

	reporter.Started("Initializing GCP connectivity")

	options := []option.ClientOption{
		option.WithCredentials(creds),
		option.WithUserAgent("open-cluster-management.io submarineraddon/v1"),
	}

	gcpClient, err := gcpClientIface.NewClient(info.ProjectID, options)
	if err != nil {
		reporter.Failed(err)
		return errors.Wrap(err, "Failed to initialize a GCP Client")
	}

	gcpCloudInfo := gcp.CloudInfo{
		ProjectID: info.ProjectID,
		InfraID:   info.InfraID,
		Region:    info.Region,
		Client:    gcpClient,
	}
	gcpCloud := gcp.NewCloud(gcpCloudInfo)
	msDeployer := ocp.NewK8sMachinesetDeployer(restMapper, dynamicClient)
	// TODO: Ideally we should be able to specify the image for GWNode, but it was seen that
	// with certain images, the instance is not coming up. Needs to be investigated further.
	gwDeployer := gcp.NewOcpGatewayDeployer(gcpCloudInfo, msDeployer, gwInstanceType, "", info.DedicatedGW, k8sClientSet)

	return function(gcpCloud, gwDeployer, reporter)
}

func ReadFromFile(filename string) (string, string, string, error) {
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return "", "", "", errors.Wrapf(err, "failed to stat file %q", filename)
	}

	metadataFile := filename

	if fileInfo.IsDir() {
		metadataFile = filepath.Join(filename, "metadata.json")
	}

	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return  "", "", "", errors.Wrapf(err, "error reading file %q", metadataFile)
	}

	var metadata struct {
		InfraID string `json:"infraID"`
		GCP     struct {
			Region    string `json:"region"`
			ProjectID string `json:"projectID"`
		} `json:"gcp"`
	}

	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return  "", "", "", errors.Wrap(err, "error unmarshalling data")
	}

	return metadata.InfraID, metadata.GCP.Region, metadata.GCP.ProjectID, nil
}

func GetCredentials(credentialsFile string) (*google.Credentials, error) {
	authJSON, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading file %q", credentialsFile)
	}

	creds, err := google.CredentialsFromJSON(context.TODO(), authJSON, dns.CloudPlatformScope)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing credentials file")
	}

	return creds, nil
}
