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
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/admiral/pkg/util"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/cloud-prepare/pkg/k8s"
	"github.com/submariner-io/cloud-prepare/pkg/ocp"
	"github.com/submariner-io/cloud-prepare/pkg/rhos"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type Config struct {
	DedicatedGateway bool
	Gateways         int
	InfraID          string
	Region           string
	ProjectID        string
	OcpMetadataFile  string
	CloudEntry       string
	GWInstanceType   string
}

// RunOn runs the given function on RHOS, supplying it with a cloud instance connected to RHOS and a reporter that writes to CLI.
// The functions makes sure that infraID and region are specified, and extracts the credentials from a secret in order to connect to RHOS.
func RunOn(restConfigProducer *restconfig.Producer, config *Config, status reporter.Interface,
	function func(api.Cloud, api.GatewayDeployer, reporter.Interface) error,
) error {
	if config.OcpMetadataFile != "" {
		var err error

		config.InfraID, config.ProjectID, err = ReadFromFile(config.OcpMetadataFile)
		config.Region = os.Getenv("OS_REGION_NAME")

		return status.Error(err, "Failed to read RHOS Cluster information from OCP metadata file")
	}

	status.Start("Retrieving RHOS credentials from your RHOS configuration")

	// Using RHOS default "openstack", if not specified
	if config.CloudEntry == "" {
		config.CloudEntry = "openstack"
	}

	opts := &clientconfig.ClientOpts{
		Cloud: config.CloudEntry,
	}

	providerClient, err := clientconfig.AuthenticatedClient(opts)
	if err != nil {
		return status.Error(err, "error initializing RHOS Client")
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

	cloudInfo := rhos.CloudInfo{
		Client:    providerClient,
		InfraID:   config.InfraID,
		Region:    config.Region,
		K8sClient: k8sClientSet,
	}
	rhosCloud := rhos.NewCloud(cloudInfo)
	msDeployer := ocp.NewK8sMachinesetDeployer(restMapper, dynamicClient)
	gwDeployer := rhos.NewOcpGatewayDeployer(cloudInfo, msDeployer, config.ProjectID, config.GWInstanceType,
		"", config.CloudEntry, config.DedicatedGateway)

	return function(rhosCloud, gwDeployer, status)
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
		RHOS    struct {
			ProjectID string `json:"projectID"`
		} `json:"rhos"`
	}

	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return "", "", errors.Wrap(err, "error unmarshalling data")
	}

	return metadata.InfraID, metadata.RHOS.ProjectID, nil
}
