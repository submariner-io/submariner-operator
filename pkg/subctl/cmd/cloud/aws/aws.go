/*
© 2021 Red Hat, Inc. and others.

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

// This package provides common functionality to run cloud prepare/cleanup on AWS
package aws

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/spf13/cobra"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	cloudprepareaws "github.com/submariner-io/cloud-prepare/pkg/aws"
	cloudutils "github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	infraIDFlag = "infra-id"
	regionFlag  = "region"
)

var (
	infraID string
	region  string
)

// AddAWSFlags adds basic flags needed by AWS
func AddAWSFlags(command *cobra.Command) {
	command.Flags().StringVar(&infraID, infraIDFlag, "", "AWS infra ID")
	command.Flags().StringVar(&region, regionFlag, "", "AWS region")
}

// RunOnAWS runs the given function on AWS, supplying it with a cloud instance connected to AWS and a reporter that writes to CLI.
// The functions makes sure that infraID and region are specified, and extracts the credentials from a secret in order to connect to AWS.
func RunOnAWS(gwInstanceType, kubeConfig, kubeContext string,
	function func(cloud api.Cloud, reporter api.Reporter) error) error {
	utils.ExpectFlag(infraIDFlag, infraID)
	utils.ExpectFlag(regionFlag, region)

	k8sConfig, err := utils.GetRestConfig(kubeConfig, kubeContext)
	utils.ExitOnError("Failed to initialize a Kubernetes config", err)

	reporter := cloudutils.NewCLIReporter()
	reporter.Started("Retrieving AWS credentials from your OpenShift installation")
	creds, err := getAWSCredentials(k8sConfig)
	if err != nil {
		reporter.Failed(err)
		return err
	}
	reporter.Succeeded("")

	reporter.Started("Establishing connection to AWS")
	awsConfig := aws.Config{
		Credentials: creds,
		Region:      aws.String(region),
	}

	awsSession, err := session.NewSession(&awsConfig)
	if err != nil {
		reporter.Failed(err)
		return err
	}
	reporter.Succeeded("")

	gwDeployer := cloudprepareaws.NewK8sMachinesetDeployer(k8sConfig)
	awsCloud := cloudprepareaws.NewCloud(gwDeployer, ec2.New(awsSession), infraID, region, gwInstanceType)
	return function(awsCloud, reporter)
}

// Retrieve AWS credentials from an OpenShift secret.
func getAWSCredentials(k8sConfig *rest.Config) (*credentials.Credentials, error) {
	kubeClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, err
	}

	credentialsSecret, err := kubeClient.CoreV1().Secrets("openshift-machine-api").Get("aws-cloud-credentials", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	accessKeyID, ok := credentialsSecret.Data["aws_access_key_id"]
	if !ok {
		return nil, errors.New("coulnd't get aws_access_key_id from the AWS credentials secret")
	}

	secretAccessKey, ok := credentialsSecret.Data["aws_secret_access_key"]
	if !ok {
		return nil, errors.New("coulnd't get aws_secret_access_key from the AWS credentials secret")
	}

	return credentials.NewStaticCredentials(string(accessKeyID), string(secretAccessKey), ""), nil
}
