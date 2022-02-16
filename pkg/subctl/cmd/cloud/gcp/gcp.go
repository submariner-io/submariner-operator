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
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/submariner-io/admiral/pkg/util"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/cloud-prepare/pkg/gcp"
	"github.com/submariner-io/cloud-prepare/pkg/k8s"
	"github.com/submariner-io/cloud-prepare/pkg/ocp"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	cloudutils "github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const (
	infraIDFlag   = "infra-id"
	regionFlag    = "region"
	projectIDFlag = "project-id"
)

type Args struct {
	cloudName       string
	InfraID         string
	Region          string
	ProjectID       string
	CredentialsFile string
	OcpMetadataFile string
}

var ClientArgs = NewArgs("")

func NewArgs(name string) *Args {
	return &Args{
		cloudName: name,
	}
}

// AddGCPFlags adds basic flags needed by GCP.
func (args *Args) AddGCPFlags(command *cobra.Command) {
	command.Flags().StringVar(&args.InfraID, infraIDFlag, "", "GCP infra ID")
	command.Flags().StringVar(&args.Region, regionFlag, "", "GCP region")
	command.Flags().StringVar(&args.ProjectID, projectIDFlag, "", "GCP project ID")
	command.Flags().StringVar(&args.OcpMetadataFile, "ocp-metadata", "",
		"OCP metadata.json file (or the directory containing it) from which to read the GCP infra ID "+
			"and region from (takes precedence over the specific flags)")

	dirname, err := os.UserHomeDir()
	if err != nil {
		exit.OnErrorWithMessage(err, "failed to find home directory")
	}

	defaultCredentials := filepath.FromSlash(fmt.Sprintf("%s/.gcp/osServiceAccount.json", dirname))
	command.Flags().StringVar(&args.CredentialsFile, "credentials", defaultCredentials, "GCP credentials configuration file")
}

// ValidateFlags if the OcpMetadataFile is provided it overrides the infra-id and region flags.
func (args *Args) ValidateFlags() {
	if args.OcpMetadataFile != "" {
		err := args.initializeFlagsFromOCPMetadata()
		exit.OnErrorWithMessage(err, "Failed to read GCP Cluster information from OCP metadata file")
	} else {
		utils.ExpectFlag(args.getFlagName(infraIDFlag), args.InfraID)
		utils.ExpectFlag(args.getFlagName(regionFlag), args.Region)
		utils.ExpectFlag(args.getFlagName(projectIDFlag), args.ProjectID)
	}
}

// RunOnGCP runs the given function on GCP, supplying it with a cloud instance connected to GCP and a reporter that writes to CLI.
// The functions makes sure that infraID and region are specified, and extracts the credentials from a secret in order to connect to GCP.
func (args *Args) RunOnGCP(restConfigProducer restconfig.Producer, gwInstanceType string, dedicatedGWNodes bool,
	function func(cloud api.Cloud, gwDeployer api.GatewayDeployer, reporter api.Reporter) error) error {
	ClientArgs.ValidateFlags()

	reporter := cloudutils.NewStatusReporter()

	reporter.Started("Initializing GCP connectivity")

	gcpCloudInfo, err := gcp.NewCloudInfoFromSettings(args.CredentialsFile, args.ProjectID, args.InfraID, args.Region)
	if err != nil {
		reporter.Failed(err)

		return errors.Wrap(err, "error loading default config")
	}

	reporter.Succeeded("")

	k8sConfig, err := restConfigProducer.ForCluster()
	exit.OnErrorWithMessage(err, "Failed to initialize a Kubernetes config")

	clientSet, err := kubernetes.NewForConfig(k8sConfig)
	exit.OnErrorWithMessage(err, "Failed to create Kubernetes client")

	k8sClientSet := k8s.NewInterface(clientSet)

	restMapper, err := util.BuildRestMapper(k8sConfig)
	exit.OnErrorWithMessage(err, "Failed to create restmapper")

	dynamicClient, err := dynamic.NewForConfig(k8sConfig)
	exit.OnErrorWithMessage(err, "Failed to create dynamic client")

	msDeployer := ocp.NewK8sMachinesetDeployer(restMapper, dynamicClient)
	// TODO: Ideally we should be able to specify the image for GWNode, but it was seen that
	// with certain images, the instance is not coming up. Needs to be investigated further.
	gwDeployer := gcp.NewOcpGatewayDeployer(*gcpCloudInfo, msDeployer, gwInstanceType, "", dedicatedGWNodes, k8sClientSet)

	exit.OnErrorWithMessage(err, "Failed to initialize a GatewayDeployer config")

	return function(gcp.NewCloud(*gcpCloudInfo), gwDeployer, reporter)
}

func (args *Args) getFlagName(flag string) string {
	if args.cloudName == "" {
		return flag
	}

	return fmt.Sprintf("%v-%v", args.cloudName, flag)
}

func (args *Args) initializeFlagsFromOCPMetadata() error {
	fileInfo, err := os.Stat(args.OcpMetadataFile)
	if err != nil {
		return errors.Wrapf(err, "failed to stat file %q", args.OcpMetadataFile)
	}

	if fileInfo.IsDir() {
		args.OcpMetadataFile = filepath.Join(args.OcpMetadataFile, "metadata.json")
	}

	data, err := os.ReadFile(args.OcpMetadataFile)
	if err != nil {
		return errors.Wrapf(err, "error reading file %q", args.OcpMetadataFile)
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

	args.InfraID = metadata.InfraID
	args.Region = metadata.GCP.Region
	args.ProjectID = metadata.GCP.ProjectID

	return nil
}

func (args *Args) getGCPCredentials() (*google.Credentials, error) {
	authJSON, err := os.ReadFile(args.CredentialsFile)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading file %q", args.CredentialsFile)
	}

	creds, err := google.CredentialsFromJSON(context.TODO(), authJSON, dns.CloudPlatformScope)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing credentials file")
	}

	return creds, nil
}
