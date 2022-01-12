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

// This package provides common functionality to run cloud prepare/cleanup on GCP Clusters.
package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/submariner-io/admiral/pkg/util"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/cloud-prepare/pkg/gcp"
	gcpClientIface "github.com/submariner-io/cloud-prepare/pkg/gcp/client"
	"github.com/submariner-io/cloud-prepare/pkg/k8s"
	"github.com/submariner-io/cloud-prepare/pkg/ocp"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	cloudutils "github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/option"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const (
	infraIDFlag   = "infra-id"
	regionFlag    = "region"
	projectIDFlag = "project-id"
)

var (
	infraID         string
	region          string
	projectID       string
	credentialsFile string
	ocpMetadataFile string
)

// AddGCPFlags adds basic flags needed by GCP.
func AddGCPFlags(command *cobra.Command) {
	command.Flags().StringVar(&infraID, infraIDFlag, "", "GCP infra ID")
	command.Flags().StringVar(&region, regionFlag, "", "GCP region")
	command.Flags().StringVar(&projectID, projectIDFlag, "", "GCP project ID")
	command.Flags().StringVar(&ocpMetadataFile, "ocp-metadata", "",
		"OCP metadata.json file (or the directory containing it) from which to read the GCP infra ID "+
			"and region from (takes precedence over the specific flags)")

	dirname, err := os.UserHomeDir()
	if err != nil {
		utils.ExitOnError("failed to find home directory", err)
	}

	defaultCredentials := filepath.FromSlash(fmt.Sprintf("%s/.gcp/osServiceAccount.json", dirname))
	command.Flags().StringVar(&credentialsFile, "credentials", defaultCredentials, "GCP credentials configuration file")
}

// RunOnGCP runs the given function on GCP, supplying it with a cloud instance connected to GCP and a reporter that writes to CLI.
// The functions makes sure that infraID and region are specified, and extracts the credentials from a secret in order to connect to GCP.
func RunOnGCP(restConfigProducer restconfig.Producer, gwInstanceType string, dedicatedGWNodes bool,
	function func(cloud api.Cloud, gwDeployer api.GatewayDeployer, reporter api.Reporter) error) error {
	if ocpMetadataFile != "" {
		err := initializeFlagsFromOCPMetadata(ocpMetadataFile)
		utils.ExitOnError("Failed to read GCP Cluster information from OCP metadata file", err)
	} else {
		utils.ExpectFlag(infraIDFlag, infraID)
		utils.ExpectFlag(regionFlag, region)
		utils.ExpectFlag(projectIDFlag, region)
	}

	reporter := cloudutils.NewStatusReporter()
	reporter.Started("Retrieving GCP credentials from your GCP configuration")

	creds, err := getGCPCredentials()
	utils.ExitOnError("Failed to get GCP credentials", err)
	reporter.Succeeded("")

	reporter.Started("Initializing GCP connectivity")

	options := []option.ClientOption{
		option.WithCredentials(creds),
		option.WithUserAgent("open-cluster-management.io submarineraddon/v1"),
	}

	gcpClient, err := gcpClientIface.NewClient(projectID, options)
	utils.ExitOnError("Failed to initialize a GCP Client", err)

	k8sConfig, err := restConfigProducer.ForCluster()
	utils.ExitOnError("Failed to initialize a Kubernetes config", err)

	clientSet, err := kubernetes.NewForConfig(k8sConfig)
	utils.ExitOnError("Failed to create Kubernetes client", err)

	k8sClientSet := k8s.NewInterface(clientSet)

	restMapper, err := util.BuildRestMapper(k8sConfig)
	utils.ExitOnError("Failed to create restmapper", err)

	dynamicClient, err := dynamic.NewForConfig(k8sConfig)
	utils.ExitOnError("Failed to create dynamic client", err)

	gcpCloudInfo := gcp.CloudInfo{
		ProjectID: projectID,
		InfraID:   infraID,
		Region:    region,
		Client:    gcpClient,
	}
	gcpCloud := gcp.NewCloud(gcpCloudInfo)
	msDeployer := ocp.NewK8sMachinesetDeployer(restMapper, dynamicClient)
	// TODO: Ideally we should be able to specify the image for GWNode, but it was seen that
	// with certain images, the instance is not coming up. Needs to be investigated further.
	gwDeployer := gcp.NewOcpGatewayDeployer(gcpCloudInfo, msDeployer, gwInstanceType, "", dedicatedGWNodes, k8sClientSet)

	utils.ExitOnError("Failed to initialize a GatewayDeployer config", err)

	return function(gcpCloud, gwDeployer, reporter)
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
		GCP     struct {
			Region    string `json:"region"`
			ProjectID string `json:"projectID"`
		} `json:"gcp"`
	}

	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling data")
	}

	infraID = metadata.InfraID
	region = metadata.GCP.Region
	projectID = metadata.GCP.ProjectID

	return nil
}

func getGCPCredentials() (*google.Credentials, error) {
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
