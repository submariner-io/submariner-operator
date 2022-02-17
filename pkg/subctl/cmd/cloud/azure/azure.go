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

package azure

import (
	"encoding/json"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/admiral/pkg/util"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/cloud-prepare/pkg/k8s"
	"github.com/submariner-io/cloud-prepare/pkg/ocp"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"k8s.io/client-go/dynamic"
	"os"
	"path/filepath"

	"github.com/submariner-io/cloud-prepare/pkg/azure"
	"k8s.io/client-go/kubernetes"
)

var (
	subscriptionID  string
	infraID         string
	region          string
	ocpMetadataFile string
	authFile        string
	authorizer      autorest.Authorizer
)

const (
	infraIDFlag = "infra-id"
	regionFlag  = "region"
)

// AddAzureFlags adds basic flags needed by Azure.
func AddAzureFlags(command *cobra.Command) {
	command.Flags().StringVar(&infraID, infraIDFlag, "", "Azure infra ID")
	command.Flags().StringVar(&region, regionFlag, "", "Azure region")
	command.Flags().StringVar(&ocpMetadataFile, "ocp-metadata", "",
		"OCP metadata.json file (or directory containing it) to read Azure infra ID and region from (Takes precedence over the flags)")
	command.Flags().StringVar(&authFile, "auth-file", "", "Azure authorization file to be used")
}

// RunOnAzure runs the given function on Azure, supplying it with a cloud instance connected to Azure and a reporter that writes to CLI.
// The function makes sure that infraID and region are specified, and extracts the credentials from a secret in order to connect to Azure.
func RunOnAzure(restConfigProducer restconfig.Producer, gwInstanceType string, status reporter.Interface,
	function func(api.Cloud, api.GatewayDeployer, reporter.Interface) error,
) error {
	if ocpMetadataFile != "" {
		err := initializeFlagsFromOCPMetadata(ocpMetadataFile)
		exit.OnErrorWithMessage(err, "Failed to read Azure information from OCP metadata file")
	} else {
		utils.ExpectFlag(infraIDFlag, infraID)
		utils.ExpectFlag(regionFlag, region)
	}

	status.Start("Retrieving Azure credentials from your Azure authorization file")

	utils.ExpectFlag("auth-file", authFile)
	err := os.Setenv("AZURE_AUTH_LOCATION", authFile)
	exit.OnErrorWithMessage(err, "Error locating authorization file")

	err = initializeFromAuthFile(authFile)
	exit.OnErrorWithMessage(err, "Failed to read authorization information from Azure authorization file")

	status.End()

	status.Start("Initializing AWS connectivity")

	// This is the most recommended of several authentication options
	// https://github.com/Azure/go-autorest/tree/master/autorest/azure/auth#more-authentication-details
	authorizer, err = auth.NewAuthorizerFromEnvironment()
	exit.OnErrorWithMessage(err, "Error getting an authorizer for Azure")

	k8sConfig, err := restConfigProducer.ForCluster()
	exit.OnErrorWithMessage(err, "Failed to initialize a Kubernetes config")

	clientSet, err := kubernetes.NewForConfig(k8sConfig.Config)
	exit.OnErrorWithMessage(err, "Failed to create Kubernetes client")

	k8sClientSet := k8s.NewInterface(clientSet)

	restMapper, err := util.BuildRestMapper(k8sConfig.Config)
	exit.OnErrorWithMessage(err, "Failed to create restmapper")

	dynamicClient, err := dynamic.NewForConfig(k8sConfig.Config)
	exit.OnErrorWithMessage(err, "Failed to create dynamic client")

	msDeployer := ocp.NewK8sMachinesetDeployer(restMapper, dynamicClient)

	cloudInfo := azure.CloudInfo{
		SubscriptionID: subscriptionID,
		InfraID:        infraID,
		Region:         region,
		BaseGroupName:  infraID + "-rg",
		Authorizer:     authorizer,
		K8sClient:      k8sClientSet,
	}

	azureCloud := azure.NewCloud(&cloudInfo)

	status.End()

	gwDeployer, err := azure.NewOcpGatewayDeployer(azureCloud, msDeployer, gwInstanceType)
	exit.OnErrorWithMessage(err, "Failed to initialize a GatewayDeployer config")

	return function(azureCloud, gwDeployer, status)
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
		Azure   struct {
			Region string `json:"region"`
		} `json:"azure"`
	}

	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling data")
	}

	infraID = metadata.InfraID
	region = metadata.Azure.Region

	return nil
}

func initializeFromAuthFile(authFile string) error {
	data, err := os.ReadFile(authFile)
	if err != nil {
		return errors.Wrapf(err, "error reading file %q", authFile)
	}

	var authInfo struct {
		ClientId       string
		ClientSecret   string
		SubscriptionId string
		TenantId       string
	}

	err = json.Unmarshal(data, &authInfo)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling data")
	}

	subscriptionID = authInfo.SubscriptionId

	if err = os.Setenv("AZURE_CLIENT_ID", authInfo.ClientId); err != nil {
		return err
	}

	if err = os.Setenv("AZURE_CLIENT_SECRET", authInfo.ClientSecret); err != nil {
		return err
	}

	if err = os.Setenv("AZURE_TENANT_ID", authInfo.TenantId); err != nil {
		return err
	}

	return nil
}
