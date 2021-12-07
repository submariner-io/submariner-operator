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

package deploy

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/stringset"
	submarinerv1a1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/internal/image"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/subctl/components"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/brokercr"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop"
	v1 "k8s.io/api/core/v1"
)

type DeployOptions struct {
	OperatorDebug       bool
	IpsecSubmFile       string
	GlobalCIDRConfigMap *v1.ConfigMap
	Repository          string
	ImageVersion        string
	BrokerNamespace     string
	BrokerSpec          submarinerv1a1.BrokerSpec
}

var ValidComponents = []string{components.ServiceDiscovery, components.Connectivity}

const brokerDetailsFilename = "broker-info.subm"

func Broker(do DeployOptions, kubeConfig, kubeContext string) error {
	status := cli.NewStatus()
	componentSet := stringset.New(do.BrokerSpec.Components...)

	if err := isValidComponents(componentSet); err != nil {
		return errors.Wrap(err, "invalid components parameter")
	}

	if do.BrokerSpec.GlobalnetEnabled {
		componentSet.Add(components.Globalnet)
	}

	if valid, err := isValidGlobalnetConfig(do); !valid {
		if err != nil {
			return errors.Wrap(err, "invalid GlobalCIDR configuration")
		}
	}

	config, err := restconfig.ForCluster(kubeConfig, kubeContext)
	if err != nil {
		return errors.Wrap(err, "the provided kubeconfig is invalid")
	}

	status.Start("Setting up broker RBAC")
	err = broker.Ensure(config, do.BrokerSpec.Components, false, do.BrokerNamespace)
	status.End(cli.CheckForError(err))
	if err != nil {
		return errors.Wrap(err, "error setting up broker RBAC")
	}

	status.Start("Deploying the Submariner operator")
	operatorImage, err := image.ForOperator(do.ImageVersion, do.Repository, nil)
	if err != nil {
		return errors.Wrap(err, "error getting Operator image")
	}
	err = submarinerop.Ensure(status, config, constants.OperatorNamespace, operatorImage, do.OperatorDebug)
	status.End(cli.CheckForError(err))
	if err != nil {
		return errors.Wrap(err, "error deploying the operator")
	}

	status.Start("Deploying the broker")
	err = brokercr.Ensure(config, do.BrokerNamespace, do.BrokerSpec)
	if err == nil {
		status.QueueSuccessMessage("The broker has been deployed")
		status.End(cli.Success)
	} else {
		status.QueueFailureMessage("Broker deployment failed")
		status.End(cli.Failure)
		return errors.Wrap(err, "error deploying the broker")
	}

	status.Start(fmt.Sprintf("Creating %s file", brokerDetailsFilename))

	// If deploy-broker is retried we will attempt to re-use the existing IPsec PSK secret
	if do.IpsecSubmFile == "" {
		if _, err := datafile.NewFromFile(brokerDetailsFilename); err == nil {
			do.IpsecSubmFile = brokerDetailsFilename
			status.QueueWarningMessage(fmt.Sprintf("Reusing IPsec PSK from existing %s", brokerDetailsFilename))
		} else {
			status.QueueSuccessMessage(fmt.Sprintf("A new IPsec PSK will be generated for %s", brokerDetailsFilename))
		}
	}

	subctlData, err := datafile.NewFromCluster(config, do.BrokerNamespace, do.IpsecSubmFile)
	if err != nil {
		return errors.Wrap(err, "error retrieving preparing the subm data file")
	}

	newFilename, err := datafile.BackupIfExists(brokerDetailsFilename)
	if err != nil {
		return errors.Wrap(err, "error backing up the brokerfile")
	}

	if newFilename != "" {
		status.QueueSuccessMessage(fmt.Sprintf("Backed up previous %s to %s", brokerDetailsFilename, newFilename))
	}

	subctlData.ServiceDiscovery = componentSet.Contains(components.ServiceDiscovery)
	subctlData.SetComponents(componentSet)

	if len(do.BrokerSpec.DefaultCustomDomains) > 0 {
		subctlData.CustomDomains = &do.BrokerSpec.DefaultCustomDomains
	}

	if do.BrokerSpec.GlobalnetEnabled {
		if err = globalnet.ValidateExistingGlobalNetworks(config, do.BrokerNamespace); err != nil {
			return errors.Wrap(err, "error validating existing globalCIDR configmap")
		}
	}

	if err = broker.CreateGlobalnetConfigMap(config, do.BrokerSpec.GlobalnetEnabled, do.BrokerSpec.GlobalnetCIDRRange,
		do.BrokerSpec.DefaultGlobalnetClusterSize, do.BrokerNamespace); err != nil {
		return errors.Wrap(err, "error creating globalCIDR configmap on Broker")
	}

	err = subctlData.WriteToFile(brokerDetailsFilename)
	status.End(cli.CheckForError(err))
	if err != nil {
		return errors.Wrap(err, "error writing the broker information")
	}
	return nil
}

func isValidComponents(componentSet stringset.Interface) error {
	validComponentSet := stringset.New(ValidComponents...)

	if componentSet.Size() < 1 {
		return fmt.Errorf("at least one component must be provided for deployment")
	}

	for _, component := range componentSet.Elements() {
		if !validComponentSet.Contains(component) {
			return fmt.Errorf("unknown component: %s", component)
		}
	}

	return nil
}

func isValidGlobalnetConfig(gnSettings DeployOptions) (bool, error) {
	var err error
	if !gnSettings.BrokerSpec.GlobalnetEnabled {
		return true, nil
	}
	gnSettings.BrokerSpec.DefaultGlobalnetClusterSize, err = globalnet.GetValidClusterSize(gnSettings.BrokerSpec.GlobalnetCIDRRange,
		gnSettings.BrokerSpec.DefaultGlobalnetClusterSize)
	if err != nil || gnSettings.BrokerSpec.DefaultGlobalnetClusterSize == 0 {
		return false, err
	}

	err = globalnet.IsValidCIDR(gnSettings.BrokerSpec.GlobalnetCIDRRange)
	return err == nil, err
}
