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

// Package rhos provides common functionality to run cloud prepare/cleanup on RHOS Clusters.
package rhos

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/cloud-prepare/pkg/k8s"
	"github.com/submariner-io/cloud-prepare/pkg/rhos"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	cloudutils "github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"k8s.io/client-go/kubernetes"
)

const (
	infraIDFlag    = "infra-id"
	regionFlag     = "region"
	projectIDFlag  = "project-id"
	cloudEntryFlag = "cloud-entry"
)

var (
	infraID         string
	region          string
	projectID       string
	ocpMetadataFile string
	cloudEntry      string
)

// AddRHOSFlags adds basic flags needed by RHOS.
func AddRHOSFlags(command *cobra.Command) {
	command.Flags().StringVar(&infraID, infraIDFlag, "", "RHOS infra ID")
	command.Flags().StringVar(&region, regionFlag, "", "RHOS region")
	command.Flags().StringVar(&projectID, projectIDFlag, "", "RHOS project ID")
	command.Flags().StringVar(&ocpMetadataFile, "ocp-metadata", "",
		"OCP metadata.json file (or the directory containing it) from which to read the RHOS infra ID "+
			"and region from (takes precedence over the specific flags)")
	command.Flags().StringVar(&cloudEntry, cloudEntryFlag, "", "the cloud entry to use")
}

// RunOnRHOS runs the given function on RHOS, supplying it with a cloud instance connected to RHOS and a reporter that writes to CLI.
// The functions makes sure that infraID and region are specified, and extracts the credentials from a secret in order to connect to RHOS.
func RunOnRHOS(restConfigProducer restconfig.Producer, function func(cloud api.Cloud, gwDeployer api.GatewayDeployer,
	reporter api.Reporter) error) error {
	if ocpMetadataFile != "" {
		err := initializeFlagsFromOCPMetadata(ocpMetadataFile)
		region = os.Getenv("OS_REGION_NAME")

		utils.ExitOnError("Failed to read RHOS Cluster information from OCP metadata file", err)
	} else {
		utils.ExpectFlag(infraIDFlag, infraID)
		utils.ExpectFlag(regionFlag, region)
		utils.ExpectFlag(projectIDFlag, projectID)
	}

	reporter := cloudutils.NewStatusReporter()
	reporter.Started("Retrieving RHOS credentials from your RHOS configuration")

	// Using RHOS default "openstack", if not specified
	if cloudEntry == "" {
		cloudEntry = "openstack"
	}

	opts := &clientconfig.ClientOpts{
		Cloud: cloudEntry,
	}

	providerClient, err := clientconfig.AuthenticatedClient(opts)

	utils.ExitOnError("Failed to initialize a RHOS Client", err)

	k8sConfig, err := restConfigProducer.ForCluster()
	utils.ExitOnError("Failed to initialize a Kubernetes config", err)

	clientSet, err := kubernetes.NewForConfig(k8sConfig)
	utils.ExitOnError("Failed to create Kubernetes client", err)

	k8sClientSet := k8s.NewInterface(clientSet)

	cloudInfo := rhos.CloudInfo{
		Client:    providerClient,
		InfraID:   infraID,
		Region:    region,
		K8sClient: k8sClientSet,
	}
	rhosCloud := rhos.NewCloud(cloudInfo)
	gwDeployer := rhos.NewOcpGatewayDeployer(cloudInfo, projectID)

	utils.ExitOnError("Failed to initialize a GatewayDeployer config", err)

	return function(rhosCloud, gwDeployer, reporter)
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
		RHOS    struct {
			ProjectID string `json:"projectID"`
		} `json:"rhos"`
	}

	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling data")
	}

	infraID = metadata.InfraID
	projectID = metadata.RHOS.ProjectID

	return nil
}
