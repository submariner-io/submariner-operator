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

	"github.com/submariner-io/submariner-operator/internal"
	"github.com/submariner-io/submariner-operator/internal/image"
	"github.com/submariner-io/submariner-operator/pkg/broker"

	"github.com/submariner-io/admiral/pkg/stringset"
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/subctl/components"
	v1 "k8s.io/api/core/v1"

	submarinerv1a1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/brokercr"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop"
)

type DeployOptions struct {
	OperatorDebug               bool
	GlobalnetEnable             bool
	IpsecSubmFile               string
	GlobalnetCIDRRange          string
	DefaultGlobalnetClusterSize uint
	ComponentArr                []string
	GlobalCIDRConfigMap         *v1.ConfigMap
	DefaultCustomDomains        []string
	Repository                  string
	ImageVersion                string
}

var ValidComponents = []string{components.ServiceDiscovery, components.Connectivity}

const brokerDetailsFilename = "broker-info.subm"

func Broker(do DeployOptions, kubeConfig, kubeContext string) error {

	status := cli.NewStatus()
	componentSet := stringset.New(do.ComponentArr...)

	if err := isValidComponents(componentSet); err != nil {
		return fmt.Errorf("invalid components parameter %s", err)
	}

	if do.GlobalnetEnable {
		componentSet.Add(components.Globalnet)
	}

	if valid, err := isValidGlobalnetConfig(do); !valid {
		if err != nil {
			return fmt.Errorf("invalid GlobalCIDR configuration %s", err)
		}
	}

	config, err := restconfig.ForCluster(kubeConfig, kubeContext)
	if err != nil {
		return fmt.Errorf("the provided kubeconfig is invalid %s", err)
	}

	status.Start("Setting up broker RBAC")
	err = broker.Ensure(config, do.ComponentArr, false)
	status.End(cli.CheckForError(err))
	if err != nil {
		return fmt.Errorf("Error setting up broker RBAC %s", err)
	}

	status.Start("Deploying the Submariner operator")
	err = submarinerop.Ensure(status, config, internal.OperatorNamespace,
		image.Operator(do.ImageVersion, do.Repository, nil), do.OperatorDebug)
	status.End(cli.CheckForError(err))
	if err != nil {
		return fmt.Errorf("Error deploying the operator %s", err)
	}

	status.Start("Deploying the broker")
	err = brokercr.Ensure(config, internal.OperatorNamespace, populateBrokerSpec(do))
	if err == nil {
		status.QueueSuccessMessage("The broker has been deployed")
		status.End(cli.Success)
	} else {
		status.QueueFailureMessage("Broker deployment failed")
		status.End(cli.Failure)
		return fmt.Errorf("Error deploying the broker %s", err)
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

	subctlData, err := datafile.NewFromCluster(config, broker.SubmarinerBrokerNamespace, do.IpsecSubmFile)
	if err != nil {
		return fmt.Errorf("Error retrieving preparing the subm data file %s", err)
	}

	newFilename, err := datafile.BackupIfExists(brokerDetailsFilename)
	if err != nil {
		return fmt.Errorf("Error backing up the brokerfile %s", err)
	}

	if newFilename != "" {
		status.QueueSuccessMessage(fmt.Sprintf("Backed up previous %s to %s", brokerDetailsFilename, newFilename))
	}

	subctlData.SetComponents(componentSet)

	if len(do.DefaultCustomDomains) > 0 {
		subctlData.CustomDomains = &do.DefaultCustomDomains
	}

	if do.GlobalnetEnable {
		if err = globalnet.ValidateExistingGlobalNetworks(config, broker.SubmarinerBrokerNamespace); err != nil {
			return fmt.Errorf("Error validating existing globalCIDR configmap %s", err)
		}
	}

	if err = broker.CreateGlobalnetConfigMap(config, do.GlobalnetEnable, do.GlobalnetCIDRRange,
		do.DefaultGlobalnetClusterSize, broker.SubmarinerBrokerNamespace); err != nil {
		return fmt.Errorf("Error creating globalCIDR configmap on Broker %s", err)
	}

	err = subctlData.WriteToFile(brokerDetailsFilename)
	status.End(cli.CheckForError(err))
	if err != nil {
		return fmt.Errorf("Error writing the broker information %s", err)
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
	if !gnSettings.GlobalnetEnable {
		return true, nil
	}
	gnSettings.DefaultGlobalnetClusterSize, err = globalnet.GetValidClusterSize(gnSettings.GlobalnetCIDRRange,
		gnSettings.DefaultGlobalnetClusterSize)
	if err != nil || gnSettings.DefaultGlobalnetClusterSize == 0 {
		return false, err
	}

	err = globalnet.IsValidCIDR(gnSettings.GlobalnetCIDRRange)
	return err == nil, err
}

func populateBrokerSpec(do DeployOptions) submarinerv1a1.BrokerSpec {
	brokerSpec := submarinerv1a1.BrokerSpec{
		GlobalnetEnabled:            do.GlobalnetEnable,
		GlobalnetCIDRRange:          do.GlobalnetCIDRRange,
		DefaultGlobalnetClusterSize: do.DefaultGlobalnetClusterSize,
		Components:                  do.ComponentArr,
		DefaultCustomDomains:        do.DefaultCustomDomains,
	}
	return brokerSpec
}
