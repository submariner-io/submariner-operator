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
package join

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/internal/image"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/servicediscoverycr"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop"
	"github.com/submariner-io/submariner-operator/pkg/version"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"github.com/pkg/errors"
)

type Options struct {
	ClusterID                     string
	ServiceCIDR                   string
	ClusterCIDR                   string
	GlobalnetCIDR                 string
	Repository                    string
	ImageVersion                  string
	NattPort                      int
	IkePort                       int
	PreferredServer               bool
	ForceUDPEncaps                bool
	ColorCodes                    string
	NatTraversal                  bool
	IgnoreRequirements            bool
	GlobalnetEnabled              bool
	IpsecDebug                    bool
	SubmarinerDebug               bool
	OperatorDebug                 bool
	LabelGateway                  bool
	LoadBalancerEnabled           bool
	CableDriver                   string
	Clienttoken                   *v1.Secret
	GlobalnetClusterSize          uint
	CustomDomains                 []string
	ImageOverrideArr              []string
	HealthCheckEnable             bool
	HealthCheckInterval           uint64
	HealthCheckMaxPacketLossCount uint64
	CorednsCustomConfigMap        string
}

var status = cli.NewStatus()

func SubmarinerCluster(jo Options, kubeContext, kubeConfig string, subctlData *datafile.SubctlData) error {
	if err := checkVersionMismatch(kubeContext, kubeConfig); err != nil {
		return errors.Wrap(err, "version mismatch error")
	}

	if err := isValidCustomCoreDNSConfig(jo.CorednsCustomConfigMap); err != nil {
		return errors.Wrap(err,"invalid Custom CoreDNS configuration")
	}

	// Missing information
	var qs = []*survey.Question{}
	config := restconfig.ClientConfig(kubeConfig, kubeContext)
	if jo.ClusterID == "" {
		rawConfig, err := config.RawConfig()
		// This will be fatal later, no point in continuing
		if err != nil {
			return errors.Wrap(err,"error connecting to the target cluster")
		}

		clusterName := restconfig.ClusterNameFromContext(rawConfig, kubeContext)
		if clusterName != nil {
			jo.ClusterID = *clusterName
		}
	}

	if valid, err := isValidClusterID(jo.ClusterID); !valid {
		fmt.Printf("Error: %s\n", err.Error())
		qs = append(qs, &survey.Question{
			Name:   "clusterID",
			Prompt: &survey.Input{Message: "What is your cluster ID?"},
			Validate: func(val interface{}) error {
				str, ok := val.(string)
				if !ok {
					return nil
				}
				_, err := isValidClusterID(str)
				return err
			},
		})
	}
	if jo.ColorCodes == "" {
		qs = append(qs, &survey.Question{
			Name:     "colorCodes",
			Prompt:   &survey.Input{Message: "What color codes should be used (e.g. \"blue\")?"},
			Validate: survey.Required,
		})
	}

	if len(qs) > 0 {
		answers := struct {
			ClusterID  string
			ColorCodes string
		}{}

		// Most likely a programming error
		if err := survey.Ask(qs, &answers); err != nil {
			return errors.Wrap(err,"error processing the answers provided")
		}

		if len(answers.ClusterID) > 0 {
			jo.ClusterID = answers.ClusterID
		}
		if len(answers.ColorCodes) > 0 {
			jo.ColorCodes = answers.ColorCodes
		}
	}

	clientConfig, err := config.ClientConfig()
	if err != nil {
		return errors.Wrap(err,"error connecting to the target cluster")
	}

	_, failedRequirements, err := version.CheckRequirements(clientConfig)
	// We display failed requirements even if an error occurred
	if len(failedRequirements) > 0 {
		fmt.Println("The target cluster fails to meet Submariner's requirements:")
		for i := range failedRequirements {
			fmt.Printf("* %s\n", (failedRequirements)[i])
		}

		if !jo.IgnoreRequirements {
			return errors.Wrap(err,"unable to check all requirements")
		}
	}
	if err != nil {
		return errors.Wrap(err,"unable to check requirements")
	}

	if subctlData.IsConnectivityEnabled() && jo.LabelGateway {
		if err := handleNodeLabels(clientConfig); err != nil {
			return errors.Wrap(err,"unable to set the gateway node up")
		}
	}

	status.Start("Discovering network details")
	networkDetails := getNetworkDetails(clientConfig)
	status.End(cli.Success)

	serviceCIDR, serviceCIDRautoDetected, err := getServiceCIDR(jo.ServiceCIDR, networkDetails)
	if err != nil {
		return errors.Wrap(err,"error determining the service CIDR")
	}

	clusterCIDR, clusterCIDRautoDetected, err := getPodCIDR(jo.ClusterCIDR, networkDetails)
	if err != nil {
		return errors.Wrap(err,"error determining the pod CIDR")
	}

	brokerAdminConfig, err := subctlData.GetBrokerAdministratorConfig()
	if err != nil {
		return errors.Wrap(err,"error retrieving broker admin config")
	}

	brokerAdminClientset, err := kubernetes.NewForConfig(brokerAdminConfig)
	if err != nil {
		return errors.Wrap(err,"error retrieving broker admin connection")
	}

	brokerNamespace := string(subctlData.ClientToken.Data["namespace"])

	netconfig := globalnet.Config{ClusterID: jo.ClusterID,
		GlobalnetCIDR:           jo.GlobalnetCIDR,
		ServiceCIDR:             serviceCIDR,
		ServiceCIDRAutoDetected: serviceCIDRautoDetected,
		ClusterCIDR:             clusterCIDR,
		ClusterCIDRAutoDetected: clusterCIDRautoDetected,
		GlobalnetClusterSize:    jo.GlobalnetClusterSize}

	if jo.GlobalnetEnabled {
		if err = AllocateAndUpdateGlobalCIDRConfigMap(brokerAdminClientset, brokerNamespace, &netconfig); err != nil {
			return errors.Wrap(err,"error Discovering multi cluster details")
		}
	}

	status.Start("Deploying the Submariner operator")

	operatorImage, err := image.ForOperator(jo.ImageVersion, jo.Repository, jo.ImageOverrideArr)
	if err != nil {
		return errors.Wrap(err, "Error overriding Operator Image")
	}
	if err = submarinerop.Ensure(status, clientConfig, constants.OperatorNamespace, operatorImage, jo.OperatorDebug); err != nil {
		status.End(cli.CheckForError(err))
		return errors.Wrap (err, "Error deploying the operator")
	}

	status.Start("Creating SA for cluster")
	jo.Clienttoken, err = broker.CreateSAForCluster(brokerAdminClientset, jo.ClusterID, brokerNamespace)
	status.End(cli.CheckForError(err))
	if err != nil {
		errors.Wrap(err, "Error creating SA for cluster")
	}

	if subctlData.IsConnectivityEnabled() {
		status.Start("Deploying Submariner")
		err = submarinercr.Ensure(clientConfig, constants.OperatorNamespace, populateSubmarinerSpec(subctlData, netconfig, jo))
		if err == nil {
			status.QueueSuccessMessage("Submariner is up and running")
			status.End(cli.Success)
		} else {
			status.QueueFailureMessage("Submariner deployment failed")
			status.End(cli.Failure)
		}

		utils.ExitOnError("Error deploying Submariner", err)
	} else if subctlData.IsServiceDiscoveryEnabled() {
		status.Start("Deploying service discovery only")
		err = servicediscoverycr.Ensure(clientConfig, constants.OperatorNamespace, populateServiceDiscoverySpec(subctlData, jo))
		if err == nil {
			status.QueueSuccessMessage("Service discovery is up and running")
			status.End(cli.Success)
		} else {
			status.QueueFailureMessage("Service discovery deployment failed")
			status.End(cli.Failure)
		}
		utils.ExitOnError("Error deploying service discovery", err)
	}
	return nil
}




