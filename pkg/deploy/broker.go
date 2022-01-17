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

	"github.com/submariner-io/admiral/pkg/stringset"
	submarinerv1a1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/component"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/internal/image"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/brokercr"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/crd"
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop"
)

type BrokerOptions struct {
	OperatorDebug   bool
	Repository      string
	ImageVersion    string
	BrokerNamespace string
	BrokerSpec      submarinerv1a1.BrokerSpec
}

var ValidComponents = []string{component.ServiceDiscovery, component.Connectivity}

func Broker(options *BrokerOptions, clientProducer client.Producer, status reporter.Interface) error {
	componentSet := stringset.New(options.BrokerSpec.Components...)

	if err := isValidComponents(componentSet); err != nil {
		return status.Error(err, "invalid components parameter")
	}

	if options.BrokerSpec.GlobalnetEnabled {
		componentSet.Add(component.Globalnet)
	}

	if err := checkGlobalnetConfig(options); err != nil {
		return status.Error(err, "invalid GlobalCIDR configuration")
	}

	err := deploy(options, status, clientProducer)
	if err != nil {
		return err
	}

	if options.BrokerSpec.GlobalnetEnabled {
		if err = globalnet.ValidateExistingGlobalNetworks(clientProducer.ForKubernetes(), options.BrokerNamespace); err != nil {
			return status.Error(err, "error validating existing globalCIDR configmap")
		}
	}

	if err = broker.CreateGlobalnetConfigMap(clientProducer.ForKubernetes(), options.BrokerSpec.GlobalnetEnabled,
		options.BrokerSpec.GlobalnetCIDRRange, options.BrokerSpec.DefaultGlobalnetClusterSize, options.BrokerNamespace); err != nil {
		return status.Error(err, "error creating globalCIDR configmap on Broker")
	}

	return nil
}

func deploy(options *BrokerOptions, status reporter.Interface, clientProducer client.Producer) error {
	status.Start("Setting up broker RBAC")

	err := broker.Ensure(crd.UpdaterFromClientSet(clientProducer.ForCRD()), clientProducer.ForKubernetes(),
		options.BrokerSpec.Components, false, options.BrokerNamespace)
	if err != nil {
		return status.Error(err, "error setting up broker RBAC")
	}

	status.End()

	status.Start("Deploying the Submariner operator")

	operatorImage, err := image.ForOperator(options.ImageVersion, options.Repository, nil)
	if err != nil {
		return status.Error(err, "error getting Operator image")
	}

	err = submarinerop.Ensure(status, clientProducer, constants.OperatorNamespace, operatorImage, options.OperatorDebug)
	if err != nil {
		return status.Error(err, "error deploying Submariner operator")
	}

	status.End()

	status.Start("Deploying the broker")

	err = brokercr.Ensure(clientProducer.ForOperator(), options.BrokerNamespace, options.BrokerSpec)
	if err != nil {
		return status.Error(err, "Broker deployment failed")
	}

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
