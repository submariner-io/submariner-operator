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
	"encoding/base64"

	"github.com/submariner-io/admiral/pkg/reporter"
	submariner "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/servicediscoverycr"
	v1 "k8s.io/api/core/v1"
)

type ServiceDiscoveryOptions struct {
	SubmarinerDebug        bool
	ClusterID              string
	CoreDNSCustomConfigMap string
	Repository             string
	ImageVersion           string
	CustomDomains          []string
}

func ServiceDiscovery(clientProducer client.Producer, options *ServiceDiscoveryOptions, brokerInfo *broker.Info, brokerSecret *v1.Secret,
	imageOverrides map[string]string, status reporter.Interface,
) error {
	serviceDiscoverySpec := populateServiceDiscoverySpec(options, brokerInfo, brokerSecret, imageOverrides)

	err := servicediscoverycr.Ensure(clientProducer.ForOperator(), constants.OperatorNamespace, serviceDiscoverySpec)
	if err != nil {
		return status.Error(err, "Service discovery deployment failed")
	}

	return nil
}

func populateServiceDiscoverySpec(options *ServiceDiscoveryOptions, brokerInfo *broker.Info, brokerSecret *v1.Secret,
	imageOverrides map[string]string,
) *submariner.ServiceDiscoverySpec {
	brokerURL := removeSchemaPrefix(brokerInfo.BrokerURL)

	serviceDiscoverySpec := submariner.ServiceDiscoverySpec{
		Repository:               options.Repository,
		Version:                  options.ImageVersion,
		BrokerK8sCA:              base64.StdEncoding.EncodeToString(brokerSecret.Data["ca.crt"]),
		BrokerK8sRemoteNamespace: string(brokerSecret.Data["namespace"]),
		BrokerK8sApiServerToken:  string(brokerSecret.Data["token"]),
		BrokerK8sApiServer:       brokerURL,
		BrokerK8sSecret:          brokerSecret.ObjectMeta.Name,
		Debug:                    options.SubmarinerDebug,
		ClusterID:                options.ClusterID,
		Namespace:                constants.SubmarinerNamespace,
		ImageOverrides:           imageOverrides,
	}

	if options.CoreDNSCustomConfigMap != "" {
		namespace, name := getCustomCoreDNSParams(options.CoreDNSCustomConfigMap)
		serviceDiscoverySpec.CoreDNSCustomConfig = &submariner.CoreDNSCustomConfig{
			ConfigMapName: name,
			Namespace:     namespace,
		}
	}

	if len(options.CustomDomains) > 0 {
		serviceDiscoverySpec.CustomDomains = options.CustomDomains
	}

	return &serviceDiscoverySpec
}
