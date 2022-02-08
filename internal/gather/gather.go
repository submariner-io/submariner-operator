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

package gather

import (
	"context"
	"fmt"

	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/component"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/brokercr"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	"github.com/submariner-io/submariner-operator/pkg/names"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Options struct {
	Type                 string
	Module               string
	Directory            string
	IncludeSensitiveData bool
}

const (
	Logs      = "logs"
	Resources = "resources"
)

var ModuleFlags = map[string]bool{
	component.Connectivity:     false,
	component.ServiceDiscovery: false,
	component.Broker:           false,
	component.Operator:         false,
}

var TypeFlags = map[string]bool{
	"logs":      false,
	"resources": false,
}

var gatherFuncs = map[string]func(string, Info) bool{
	component.Connectivity:     gatherConnectivity,
	component.ServiceDiscovery: gatherDiscovery,
	component.Broker:           gatherBroker,
	component.Operator:         gatherOperator,
}

func Data(clusterInfo *cluster.Info, status reporter.Interface, options Options) bool {
	gatherDataByCluster(clusterInfo, status, options)

	fmt.Printf("Files are stored under directory %q\n", options.Directory)

	return true
}

func gatherDataByCluster(clusterInfo *cluster.Info, status reporter.Interface, options Options) {
	var err error
	clusterName := clusterInfo.Name

	fmt.Printf("Gathering information from cluster %q\n", clusterName)

	info := Info{
		ClusterName:          clusterName,
		DirName:              options.Directory,
		IncludeSensitiveData: options.IncludeSensitiveData,
		Summary:              &Summary{},
		ClientProducer:       clusterInfo.ClientProducer,
		Submariner:           clusterInfo.Submariner,
	}

	info.ServiceDiscovery, err = clusterInfo.ClientProducer.ForOperator().SubmarinerV1alpha1().ServiceDiscoveries(constants.OperatorNamespace).
		Get(context.TODO(), names.ServiceDiscoveryCrName, metav1.GetOptions{})
	if err != nil {
		info.ServiceDiscovery = nil

		if !apierrors.IsNotFound(err) {
			status.Failure("Error getting ServiceDiscovery resource: %s", err)
			return
		}
	}

	for module, ok := range ModuleFlags {
		if ok {
			for dataType, ok := range TypeFlags {
				if ok {
					info.Status = cli.NewReporter()
					info.Status.Start("Gathering %s %s", module, dataType)

					if gatherFuncs[module](dataType, info) {
						info.Status.End()
					}
				}
			}
		}
	}

	gatherClusterSummary(&info)
}

// nolint:gocritic // hugeParam: info - purposely passed by value.
func gatherConnectivity(dataType string, info Info) bool {
	if info.Submariner == nil {
		info.Status.Warning("The Submariner connectivity components are not installed")
		return true
	}

	switch dataType {
	case Logs:
		gatherGatewayPodLogs(&info)
		gatherRouteAgentPodLogs(&info)
		gatherGlobalnetPodLogs(&info)
		gatherNetworkPluginSyncerPodLogs(&info)
	case Resources:
		gatherCNIResources(&info, info.Submariner.Status.NetworkPlugin)
		gatherCableDriverResources(&info, info.Submariner.Spec.CableDriver)
		gatherOVNResources(&info, info.Submariner.Status.NetworkPlugin)
		gatherEndpoints(&info, constants.SubmarinerNamespace)
		gatherClusters(&info, constants.SubmarinerNamespace)
		gatherGateways(&info, constants.SubmarinerNamespace)
		gatherClusterGlobalEgressIPs(&info)
		gatherGlobalEgressIPs(&info)
		gatherGlobalIngressIPs(&info)
	default:
		return false
	}

	return true
}

// nolint:gocritic // hugeParam: info - purposely passed by value.
func gatherDiscovery(dataType string, info Info) bool {
	if info.ServiceDiscovery == nil {
		info.Status.Warning("The Submariner service discovery components are not installed")
		return true
	}

	switch dataType {
	case Logs:
		gatherServiceDiscoveryPodLogs(&info)
		gatherCoreDNSPodLogs(&info)
	case Resources:
		gatherServiceExports(&info, corev1.NamespaceAll)
		gatherServiceImports(&info, corev1.NamespaceAll)
		gatherEndpointSlices(&info, corev1.NamespaceAll)
		gatherConfigMapLighthouseDNS(&info, constants.SubmarinerNamespace)
		gatherConfigMapCoreDNS(&info)
	default:
		return false
	}

	return true
}

// nolint:gocritic // hugeParam: info - purposely passed by value.
func gatherBroker(dataType string, info Info) bool {
	switch dataType {
	case Resources:
		brokerRestConfig, brokerNamespace, err := restconfig.ForBroker(info.Submariner, info.ServiceDiscovery)
		if err != nil {
			info.Status.Failure("Error getting the broker's rest config: %s", err)
			return true
		}

		if brokerRestConfig != nil {
			info.ClientProducer, err = client.NewProducerFromRestConfig(brokerRestConfig)
			if err != nil {
				info.Status.Failure("Error creating broker client Producer: %s", err)
				return true
			}
		} else {
			_, err = info.ClientProducer.ForOperator().SubmarinerV1alpha1().Brokers(constants.OperatorNamespace).Get(
				context.TODO(), brokercr.Name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return false
			}

			if err != nil {
				info.Status.Failure("Error getting the Broker resource: %s", err)
				return true
			}

			brokerNamespace = metav1.NamespaceAll
		}

		info.ClusterName = "broker"

		// The broker's ClusterRole used by member clusters only allows the below resources to be queried
		gatherEndpoints(&info, brokerNamespace)
		gatherClusters(&info, brokerNamespace)
		gatherEndpointSlices(&info, brokerNamespace)
		gatherServiceImports(&info, brokerNamespace)
	default:
		return false
	}

	return true
}

// nolint:gocritic // hugeParam: info - purposely passed by value.
func gatherOperator(dataType string, info Info) bool {
	switch dataType {
	case Logs:
		gatherSubmarinerOperatorPodLogs(&info)
	case Resources:
		gatherSubmariners(&info, constants.SubmarinerNamespace)
		gatherServiceDiscoveries(&info, constants.SubmarinerNamespace)
		gatherSubmarinerOperatorDeployment(&info, constants.SubmarinerNamespace)
		gatherGatewayDaemonSet(&info, constants.SubmarinerNamespace)
		gatherRouteAgentDaemonSet(&info, constants.SubmarinerNamespace)
		gatherGlobalnetDaemonSet(&info, constants.SubmarinerNamespace)
		gatherNetworkPluginSyncerDeployment(&info, constants.SubmarinerNamespace)
		gatherLighthouseAgentDeployment(&info, constants.SubmarinerNamespace)
		gatherLighthouseCoreDNSDeployment(&info, constants.SubmarinerNamespace)
	default:
		return false
	}

	return true
}
