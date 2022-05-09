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
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/admiral/pkg/util"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/cloud-prepare/pkg/gcp"
	gcpClientIface "github.com/submariner-io/cloud-prepare/pkg/gcp/client"
	"github.com/submariner-io/cloud-prepare/pkg/k8s"
	"github.com/submariner-io/cloud-prepare/pkg/ocp"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/option"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type Config struct {
	DedicatedGateway bool
	Gateways         int
	InfraID          string
	Region           string
	ProjectID        string
	CredentialsFile  string
	OcpMetadataFile  string
	GWInstanceType   string
}

// RunOn runs the given function on GCP, supplying it with a cloud instance connected to GCP and a reporter that writes to CLI.
// The functions makes sure that infraID and region are specified, and extracts the credentials from a secret in order to connect to GCP.
func RunOn(restConfigProducer *restconfig.Producer, config *Config, status reporter.Interface,
	function func(api.Cloud, api.GatewayDeployer, reporter.Interface) error,
) error {
	var err error
	if config.OcpMetadataFile != "" {
		config.InfraID, config.Region, config.ProjectID, err = ReadFromFile(config.OcpMetadataFile)
		return status.Error(err, "Failed to read GCP Cluster information from OCP metadata file")
	}

	status.Start("Retrieving GCP credentials from your GCP configuration")

	creds, err := getCredentials(config.CredentialsFile)
	if err != nil {
		return status.Error(err, "error retrieving GCP credentials")
	}

	status.End()

	status.Start("Initializing GCP connectivity")

	options := []option.ClientOption{
		option.WithCredentials(creds),
		option.WithUserAgent("open-cluster-management.io submarineraddon/v1"),
	}

	gcpClient, err := gcpClientIface.NewClient(config.ProjectID, options)
	if err != nil {
		return status.Error(err, "error initializing a GCP Client")
	}

	status.End()

	k8sConfig, err := restConfigProducer.ForCluster()
	if err != nil {
		return status.Error(err, "error initializing Kubernetes config")
	}

	clientSet, err := kubernetes.NewForConfig(k8sConfig.Config)
	if err != nil {
		return status.Error(err, "error creating Kubernetes client")
	}

	k8sClientSet := k8s.NewInterface(clientSet)

	restMapper, err := util.BuildRestMapper(k8sConfig.Config)
	if err != nil {
		return status.Error(err, "error creating REST mapper")
	}

	dynamicClient, err := dynamic.NewForConfig(k8sConfig.Config)
	if err != nil {
		return status.Error(err, "error creating dynamic client")
	}

	gcpCloudInfo := gcp.CloudInfo{
		ProjectID: config.ProjectID,
		InfraID:   config.InfraID,
		Region:    config.Region,
		Client:    gcpClient,
	}
	gcpCloud := gcp.NewCloud(gcpCloudInfo)
	msDeployer := ocp.NewK8sMachinesetDeployer(restMapper, dynamicClient)
	// TODO: Ideally we should be able to specify the image for GWNode, but it was seen that
	// with certain images, the instance is not coming up. Needs to be investigated further.
	gwDeployer := gcp.NewOcpGatewayDeployer(gcpCloudInfo, msDeployer, config.GWInstanceType, "", config.DedicatedGateway, k8sClientSet)

	return function(gcpCloud, gwDeployer, status)
}

func ReadFromFile(metadataFile string) (string, string, string, error) {
	fileInfo, err := os.Stat(metadataFile)
	if err != nil {
		return "", "", "", errors.Wrapf(err, "failed to stat file %q", metadataFile)
	}

	if fileInfo.IsDir() {
		metadataFile = filepath.Join(metadataFile, "metadata.json")
	}

	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return "", "", "", errors.Wrapf(err, "error reading file %q", metadataFile)
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
		return "", "", "", errors.Wrap(err, "error unmarshalling data")
	}

	return metadata.InfraID, metadata.GCP.Region, metadata.GCP.ProjectID, nil
}

func getCredentials(credentialsFile string) (*google.Credentials, error) {
	authJSON, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading file %q", credentialsFile)
	}

	creds, err := google.CredentialsFromJSON(context.TODO(), authJSON, dns.CloudPlatformScope)

	return creds, errors.Wrapf(err, "error parsing credentials file")
}
