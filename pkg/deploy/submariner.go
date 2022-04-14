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

	"github.com/submariner-io/admiral/pkg/reporter"
	submariner "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"
	"github.com/submariner-io/submariner-operator/pkg/secret"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	v1 "k8s.io/api/core/v1"
)

type SubmarinerOptions struct {
	PreferredServer               bool
	ForceUDPEncaps                bool
	NATTraversal                  bool
	IPSecDebug                    bool
	SubmarinerDebug               bool
	LoadBalancerEnabled           bool
	HealthCheckEnabled            bool
	MultiActiveGatewayEnabled     bool
	NATTPort                      int
	HealthCheckInterval           uint64
	HealthCheckMaxPacketLossCount uint64
	ClusterID                     string
	CableDriver                   string
	CoreDNSCustomConfigMap        string
	Repository                    string
	ImageVersion                  string
	ServiceCIDR                   string
	ClusterCIDR                   string
	CustomDomains                 []string
}

func Submariner(clientProducer client.Producer, options *SubmarinerOptions, brokerInfo *broker.Info, brokerSecret *v1.Secret,
	netconfig globalnet.Config, imageOverrides map[string]string, status reporter.Interface,
) error {
	pskSecret, err := secret.Ensure(clientProducer.ForKubernetes(), constants.OperatorNamespace, brokerInfo.IPSecPSK)
	if err != nil {
		return status.Error(err, "Error creating PSK secret for cluster")
	}

	submarinerSpec := populateSubmarinerSpec(options, brokerInfo, brokerSecret, pskSecret, netconfig, imageOverrides)

	err = submarinercr.Ensure(clientProducer.ForOperator(), constants.OperatorNamespace, submarinerSpec)
	if err != nil {
		return status.Error(err, "Submariner deployment failed")
	}

	return nil
}

func populateSubmarinerSpec(options *SubmarinerOptions, brokerInfo *broker.Info, brokerSecret *v1.Secret, pskSecret *v1.Secret,
	netconfig globalnet.Config, imageOverrides map[string]string,
) *submariner.SubmarinerSpec {
	brokerURL := removeSchemaPrefix(brokerInfo.BrokerURL)

	// For backwards compatibility, the connection information is populated through the secret and individual components
	// TODO skitt This will be removed in the release following 0.12
	submarinerSpec := &submariner.SubmarinerSpec{
		Repository:               getImageRepo(options.Repository),
		Version:                  getImageVersion(options.ImageVersion),
		CeIPSecNATTPort:          options.NATTPort,
		CeIPSecDebug:             options.IPSecDebug,
		CeIPSecForceUDPEncaps:    options.ForceUDPEncaps,
		CeIPSecPreferredServer:   options.PreferredServer,
		CeIPSecPSK:               base64.StdEncoding.EncodeToString(brokerInfo.IPSecPSK.Data["psk"]),
		CeIPSecPSKSecret:         pskSecret.ObjectMeta.Name,
		BrokerK8sCA:              base64.StdEncoding.EncodeToString(brokerSecret.Data["ca.crt"]),
		BrokerK8sRemoteNamespace: string(brokerSecret.Data["namespace"]),
		BrokerK8sApiServerToken:  string(brokerSecret.Data["token"]),
		BrokerK8sApiServer:       brokerURL,
		BrokerK8sSecret:          brokerSecret.ObjectMeta.Name,
		Broker:                   "k8s",
		NatEnabled:               options.NATTraversal,
		Debug:                    options.SubmarinerDebug,
		ClusterID:                options.ClusterID,
		ServiceCIDR:              options.ServiceCIDR,
		ClusterCIDR:              options.ClusterCIDR,
		Namespace:                constants.SubmarinerNamespace,
		CableDriver:              options.CableDriver,
		ServiceDiscoveryEnabled:  brokerInfo.IsServiceDiscoveryEnabled(),
		ImageOverrides:           imageOverrides,
		LoadBalancerEnabled:      options.LoadBalancerEnabled,
		ConnectionHealthCheck: &submariner.HealthCheckSpec{
			Enabled:            options.HealthCheckEnabled,
			IntervalSeconds:    options.HealthCheckInterval,
			MaxPacketLossCount: options.HealthCheckMaxPacketLossCount,
		},
	}
	if netconfig.GlobalCIDR != "" {
		submarinerSpec.GlobalCIDR = netconfig.GlobalCIDR
	}

	if options.CoreDNSCustomConfigMap != "" {
		namespace, name := getCustomCoreDNSParams(options.CoreDNSCustomConfigMap)
		submarinerSpec.CoreDNSCustomConfig = &submariner.CoreDNSCustomConfig{
			ConfigMapName: name,
			Namespace:     namespace,
		}
	}

	if len(options.CustomDomains) > 0 {
		submarinerSpec.CustomDomains = options.CustomDomains
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
		brokerURL = brokerURL[idx+3:]
	}

	return brokerURL
}
