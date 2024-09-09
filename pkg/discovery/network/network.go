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

package network

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/names"
	"k8s.io/apimachinery/pkg/types"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterNetwork struct {
	PodCIDRs         []string
	ServiceCIDRs     []string
	NetworkPlugin    string
	GlobalCIDR       string
	ClustersetIPCIDR string
	PluginSettings   map[string]string
}

func (cn *ClusterNetwork) Show() {
	if cn == nil {
		fmt.Println("    No network details discovered")
	} else {
		fmt.Printf("        Network plugin:  %s\n", cn.NetworkPlugin)
		fmt.Printf("        Service CIDRs:   %v\n", cn.ServiceCIDRs)
		fmt.Printf("        Cluster CIDRs:   %v\n", cn.PodCIDRs)

		if cn.GlobalCIDR != "" {
			fmt.Printf("        Global CIDR:     %v\n", cn.GlobalCIDR)
		}

		if cn.ClustersetIPCIDR != "" {
			fmt.Printf("        ClustersetIP CIDR:     %v\n", cn.ClustersetIPCIDR)
		}
	}
}

func (cn *ClusterNetwork) Log(logger logr.Logger) {
	logger.Info("Discovered K8s network details",
		"plugin", cn.NetworkPlugin,
		"clusterCIDRs", cn.PodCIDRs,
		"serviceCIDRs", cn.ServiceCIDRs)
}

func (cn *ClusterNetwork) IsComplete() bool {
	return cn != nil && len(cn.ServiceCIDRs) > 0 && len(cn.PodCIDRs) > 0
}

func Discover(ctx context.Context, client controllerClient.Client, operatorNamespace string) (*ClusterNetwork, error) {
	discovery, err := networkPluginsDiscovery(ctx, client)
	if discovery != nil {
		// If the info we got from the non-generic plugins is incomplete
		// try to complete with the generic discovery mechanisms
		if !discovery.IsComplete() {
			var genericNet *ClusterNetwork

			genericNet, err = discoverGenericNetwork(ctx, client)
			if genericNet != nil {
				if len(discovery.ServiceCIDRs) == 0 {
					discovery.ServiceCIDRs = genericNet.ServiceCIDRs
				}

				if len(discovery.PodCIDRs) == 0 {
					discovery.PodCIDRs = genericNet.PodCIDRs
				}
			}
		}
	} else {
		// If nothing specific was discovered, use the generic discovery
		discovery, err = discoverGenericNetwork(ctx, client)
	}

	if discovery != nil {
		globalCIDR, clustersetIPCIDR, _ := getCIDRs(ctx, client, operatorNamespace)
		discovery.GlobalCIDR = globalCIDR
		discovery.ClustersetIPCIDR = clustersetIPCIDR
	}

	return discovery, err
}

type pluginDiscoveryFn func(context.Context, controllerClient.Client) (*ClusterNetwork, error)

var discoverFunctions = []pluginDiscoveryFn{
	discoverOpenShift4Network,
	discoverOvnKubernetesNetwork,
	discoverWeaveNetwork,
	discoverCanalFlannelNetwork,
	discoverCalicoNetwork,
	discoverFlannelNetwork,
	discoverKindNetwork,
}

//nolint:nilnil // Intentional as the purpose is to discover.
func networkPluginsDiscovery(ctx context.Context, client controllerClient.Client) (*ClusterNetwork, error) {
	for _, function := range discoverFunctions {
		network, err := function(ctx, client)
		if err != nil || network != nil {
			return network, err
		}
	}

	return nil, nil
}

func getCIDRs(ctx context.Context, operatorClient controllerClient.Client, operatorNamespace string) (string, string, error) {
	if operatorClient == nil {
		return "", "", nil
	}

	existingCfg := v1alpha1.Submariner{}

	err := operatorClient.Get(ctx, types.NamespacedName{Namespace: operatorNamespace, Name: names.SubmarinerCrName}, &existingCfg)
	if err != nil {
		return "", "", errors.Wrap(err, "error retrieving Submariner resource")
	}

	globalCIDR := existingCfg.Spec.GlobalCIDR
	clustersetIPCIDR := existingCfg.Spec.ClustersetIPCIDR

	return globalCIDR, clustersetIPCIDR, nil
}
