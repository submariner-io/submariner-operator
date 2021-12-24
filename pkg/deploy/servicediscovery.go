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

	submariner "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/servicediscoverycr"
	v1 "k8s.io/api/core/v1"
)

func ServiceDiscovery(clientProducer client.Producer, jo *WithJoinOptions, brokerInfo *broker.Info, brokerSecret *v1.Secret,
	imageOverrides map[string]string, status reporter.Interface) error {
	serviceDiscoverySpec := populateServiceDiscoverySpec(jo, brokerInfo, brokerSecret, imageOverrides)

	err := servicediscoverycr.Ensure(clientProducer.ForOperator(), constants.OperatorNamespace, serviceDiscoverySpec)
	if err != nil {
		return status.Error(err, "Service discovery deployment failed")
	}

	return nil
}

func populateServiceDiscoverySpec(jo *WithJoinOptions, brokerInfo *broker.Info, brokerSecret *v1.Secret,
	imageOverrides map[string]string) *submariner.ServiceDiscoverySpec {
	brokerURL := removeSchemaPrefix(brokerInfo.BrokerURL)

	if jo.CustomDomains == nil && brokerInfo.CustomDomains != nil {
		jo.CustomDomains = *brokerInfo.CustomDomains
	}

	serviceDiscoverySpec := submariner.ServiceDiscoverySpec{
		Repository:               jo.Repository,
		Version:                  jo.ImageVersion,
		BrokerK8sCA:              base64.StdEncoding.EncodeToString(brokerSecret.Data["ca.crt"]),
		BrokerK8sRemoteNamespace: string(brokerSecret.Data["namespace"]),
		BrokerK8sApiServerToken:  string(brokerSecret.Data["token"]),
		BrokerK8sApiServer:       brokerURL,
		BrokerK8sSecret:          brokerSecret.ObjectMeta.Name,
		Debug:                    jo.SubmarinerDebug,
		ClusterID:                jo.ClusterID,
		Namespace:                constants.SubmarinerNamespace,
		ImageOverrides:           imageOverrides,
	}

	if jo.CorednsCustomConfigMap != "" {
		namespace, name := getCustomCoreDNSParams(jo.CorednsCustomConfigMap)
		serviceDiscoverySpec.CoreDNSCustomConfig = &submariner.CoreDNSCustomConfig{
			ConfigMapName: name,
			Namespace:     namespace,
		}
	}

	if len(jo.CustomDomains) > 0 {
		serviceDiscoverySpec.CustomDomains = jo.CustomDomains
	}

	return &serviceDiscoverySpec
}
