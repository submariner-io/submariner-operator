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
	"github.com/submariner-io/submariner-operator/internal/component"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/internal/image"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/brokercr"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

type BrokerOptions struct {
	OperatorDebug       bool
	IpsecSubmFile       string
	GlobalCIDRConfigMap *v1.ConfigMap
	Repository          string
	ImageVersion        string
	BrokerNamespace     string
	BrokerSpec          submarinerv1a1.BrokerSpec
}

var ValidComponents = []string{component.ServiceDiscovery, component.Connectivity}

const brokerDetailsFilename = "broker-info.subm"

// Ignoring th cyclic complexity of Broker function because it is being refactored in
// https://github.com/submariner-io/submariner-operator/pull/1717.
//gocyclo:ignore
func Broker(options *BrokerOptions, restConfigProducer restconfig.Producer, status reporter.Interface) error {
	componentSet := stringset.New(options.BrokerSpec.Components...)

	if err := isValidComponents(componentSet); err != nil {
		return errors.Wrap(err, "invalid components parameter")
	}

	if options.BrokerSpec.GlobalnetEnabled {
		componentSet.Add(component.Globalnet)
	}

	if err := checkGlobalnetConfig(options); err != nil {
		return errors.Wrap(err, "invalid GlobalCIDR configuration")
	}

	config, err := restConfigProducer.ForCluster()
	if err != nil {
		return errors.Wrap(err, "the provided kubeconfig is invalid")
	}

	if err := deploy(options, status, config); err != nil {
		return err
	}

	status.Start("Creating %s file", brokerDetailsFilename)

	// If deploy-broker is retried we will attempt to re-use the existing IPsec PSK secret
	if options.IpsecSubmFile == "" {
		if _, err := datafile.NewFromFile(brokerDetailsFilename); err == nil {
			options.IpsecSubmFile = brokerDetailsFilename
			status.Warning("Reusing IPsec PSK from existing %s", brokerDetailsFilename)
		} else {
			status.Success("A new IPsec PSK will be generated for %s", brokerDetailsFilename)
		}
	}

	subctlData, err := datafile.NewFromCluster(config, options.BrokerNamespace, options.IpsecSubmFile)
	if err != nil {
		return status.Error(err, "error retrieving preparing the subm data file")
	}

	newFilename, err := datafile.BackupIfExists(brokerDetailsFilename)
	if err != nil {
		return status.Error(err, "error backing up the brokerfile")
	}

	if newFilename != "" {
		status.Success("Backed up previous %s to %s", brokerDetailsFilename, newFilename)
	}

	subctlData.ServiceDiscovery = componentSet.Contains(component.ServiceDiscovery)
	subctlData.SetComponents(componentSet)

	if len(options.BrokerSpec.DefaultCustomDomains) > 0 {
		subctlData.CustomDomains = &options.BrokerSpec.DefaultCustomDomains
	}

	if options.BrokerSpec.GlobalnetEnabled {
		if err = globalnet.ValidateExistingGlobalNetworks(config, options.BrokerNamespace); err != nil {
			return errors.Wrap(err, "error validating existing globalCIDR configmap")
		}
	}

	if err = broker.CreateGlobalnetConfigMap(config, options.BrokerSpec.GlobalnetEnabled, options.BrokerSpec.GlobalnetCIDRRange,
		options.BrokerSpec.DefaultGlobalnetClusterSize, options.BrokerNamespace); err != nil {
		return errors.Wrap(err, "error creating globalCIDR configmap on Broker")
	}

	err = subctlData.WriteToFile(brokerDetailsFilename)
	if err != nil {
		return status.Error(err, "error writing the broker information")
	}

	status.End()

	return nil
}

func deploy(options *BrokerOptions, status reporter.Interface, config *rest.Config) error {
	status.Start("Setting up broker RBAC")

	err := broker.Ensure(config, options.BrokerSpec.Components, false, options.BrokerNamespace)
	if err != nil {
		return status.Error(err, "error setting up broker RBAC")
	}

	status.End()

	status.Start("Deploying the Submariner operator")

	operatorImage, err := image.ForOperator(options.ImageVersion, options.Repository, nil)
	if err != nil {
		return status.Error(err, "error getting Operator image")
	}

	err = submarinerop.Ensure(status, config, constants.OperatorNamespace, operatorImage, options.OperatorDebug)
	if err != nil {
		return status.Error(err, "error deploying Submariner operator")
	}

	status.End()

	status.Start("Deploying the broker")

	err = brokercr.Ensure(config, options.BrokerNamespace, options.BrokerSpec)
	if err != nil {
		return status.Error(err, "Broker deployment failed")
	}

	status.Success("The broker has been deployed")
	status.End()

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

// nolint:wrapcheck // No need to wrap errors here.
func checkGlobalnetConfig(options *BrokerOptions) error {
	var err error

	if !options.BrokerSpec.GlobalnetEnabled {
		return nil
	}

	options.BrokerSpec.DefaultGlobalnetClusterSize, err = globalnet.GetValidClusterSize(options.BrokerSpec.GlobalnetCIDRRange,
		options.BrokerSpec.DefaultGlobalnetClusterSize)
	if err != nil {
		return err
	}

	return globalnet.IsValidCIDR(options.BrokerSpec.GlobalnetCIDRRange)
}
