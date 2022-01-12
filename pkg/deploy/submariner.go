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
	"strings"

	submariner "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/secret"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	v1 "k8s.io/api/core/v1"
)

func Submariner(clientProducer client.Producer, jo *WithJoinOptions, brokerInfo *broker.Info, brokerSecret *v1.Secret,
	netconfig globalnet.Config, imageOverrides map[string]string, status reporter.Interface) error {
	pskSecret, err := secret.Ensure(clientProducer.ForKubernetes(), constants.OperatorNamespace, brokerInfo.IPSecPSK)
	if err != nil {
		return status.Error(err, "Error creating PSK secret for cluster")
	}

	submarinerSpec := populateSubmarinerSpec(jo, brokerInfo, brokerSecret, pskSecret, netconfig, imageOverrides)

	err = submarinercr.Ensure(clientProducer.ForOperator(), constants.OperatorNamespace, submarinerSpec)
	if err != nil {
		return status.Error(err, "Submariner deployment failed")
	}

	return nil
}

func populateSubmarinerSpec(jo *WithJoinOptions, brokerInfo *broker.Info, brokerSecret *v1.Secret, pskSecret *v1.Secret,
	netconfig globalnet.Config, imageOverrides map[string]string) *submariner.SubmarinerSpec {
	brokerURL := removeSchemaPrefix(brokerInfo.BrokerURL)

	if jo.CustomDomains == nil && brokerInfo.CustomDomains != nil {
		jo.CustomDomains = *brokerInfo.CustomDomains
	}

	// For backwards compatibility, the connection information is populated through the secret and individual components
	// TODO skitt This will be removed in the release following 0.12
	submarinerSpec := &submariner.SubmarinerSpec{
		Repository:               getImageRepo(jo.Repository),
		Version:                  getImageVersion(jo.ImageVersion),
		CeIPSecNATTPort:          jo.NattPort,
		CeIPSecIKEPort:           jo.IkePort,
		CeIPSecDebug:             jo.IpsecDebug,
		CeIPSecForceUDPEncaps:    jo.ForceUDPEncaps,
		CeIPSecPreferredServer:   jo.PreferredServer,
		CeIPSecPSK:               base64.StdEncoding.EncodeToString(brokerInfo.IPSecPSK.Data["psk"]),
		CeIPSecPSKSecret:         pskSecret.ObjectMeta.Name,
		BrokerK8sCA:              base64.StdEncoding.EncodeToString(brokerSecret.Data["ca.crt"]),
		BrokerK8sRemoteNamespace: string(brokerSecret.Data["namespace"]),
		BrokerK8sApiServerToken:  string(brokerSecret.Data["token"]),
		BrokerK8sApiServer:       brokerURL,
		BrokerK8sSecret:          brokerSecret.ObjectMeta.Name,
		Broker:                   "k8s",
		NatEnabled:               jo.NatTraversal,
		Debug:                    jo.SubmarinerDebug,
		ColorCodes:               jo.ColorCodes,
		ClusterID:                jo.ClusterID,
		ServiceCIDR:              netconfig.ServiceCIDR,
		ClusterCIDR:              netconfig.ClusterCIDR,
		Namespace:                constants.SubmarinerNamespace,
		CableDriver:              jo.CableDriver,
		ServiceDiscoveryEnabled:  brokerInfo.IsServiceDiscoveryEnabled(),
		ImageOverrides:           imageOverrides,
		LoadBalancerEnabled:      jo.LoadBalancerEnabled,
		ConnectionHealthCheck: &submariner.HealthCheckSpec{
			Enabled:            jo.HealthCheckEnable,
			IntervalSeconds:    jo.HealthCheckInterval,
			MaxPacketLossCount: jo.HealthCheckMaxPacketLossCount,
		},
	}
	if netconfig.GlobalnetCIDR != "" {
		submarinerSpec.GlobalCIDR = netconfig.GlobalnetCIDR
	}

	if jo.CorednsCustomConfigMap != "" {
		namespace, name := getCustomCoreDNSParams(jo.CorednsCustomConfigMap)
		submarinerSpec.CoreDNSCustomConfig = &submariner.CoreDNSCustomConfig{
			ConfigMapName: name,
			Namespace:     namespace,
		}
	}

	if len(jo.CustomDomains) > 0 {
		submarinerSpec.CustomDomains = jo.CustomDomains
	}

	return submarinerSpec
}

func getImageVersion(imageVersion string) string {
	if imageVersion == "" {
		return submariner.DefaultSubmarinerOperatorVersion
	}

	return imageVersion
}

func getImageRepo(imagerepo string) string {
	if imagerepo == "" {
		return submariner.DefaultRepo
	}

	return imagerepo
}

func getCustomCoreDNSParams(corednsCustomConfigMap string) (namespace, name string) {
	if corednsCustomConfigMap != "" {
		name = corednsCustomConfigMap

		paramList := strings.Split(corednsCustomConfigMap, "/")
		if len(paramList) > 1 {
			namespace = paramList[0]
			name = paramList[1]
		}
	}

	return namespace, name
}

func removeSchemaPrefix(brokerURL string) string {
	if idx := strings.Index(brokerURL, "://"); idx >= 0 {
		// Submariner doesn't work with a schema prefix
		brokerURL = brokerURL[(idx + 3):]
	}

	return brokerURL
}
