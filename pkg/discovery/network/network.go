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
	submariner "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/names"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterNetwork struct {
	PodCIDRs       []string
	ServiceCIDRs   []string
	NetworkPlugin  string
	GlobalCIDR     string
	PluginSettings map[string]string
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

func Discover(dynClient dynamic.Interface, clientSet kubernetes.Interface, operatorClient client.Client,
	operatorNamespace string,
) (*ClusterNetwork, error) {
	discovery, err := networkPluginsDiscovery(dynClient, clientSet)
	if err != nil {
		return nil, err
	}

	if discovery != nil {
		// TODO: The other branch of this if will not try to find the globalCIDRs
		globalCIDR, _ := getGlobalCIDRs(operatorClient, operatorNamespace)
		discovery.GlobalCIDR = globalCIDR

		if discovery.IsComplete() {
			return discovery, nil
		}

		// If the info we got from the non-generic plugins is incomplete
		// try to complete with the generic discovery mechanisms
		if len(discovery.ServiceCIDRs) == 0 || len(discovery.PodCIDRs) == 0 {
			genericNet, err := discoverGenericNetwork(clientSet)
			if err != nil {
				return nil, err
			}

			if genericNet != nil {
				if len(discovery.ServiceCIDRs) == 0 {
					discovery.ServiceCIDRs = genericNet.ServiceCIDRs
				}

				if len(discovery.PodCIDRs) == 0 {
					discovery.PodCIDRs = genericNet.PodCIDRs
				}
			}
		}

		return discovery, nil
	}

	// If nothing specific was discovered, use the generic discovery
	return discoverGenericNetwork(clientSet)
}

// nolint:nilnil // Intentional as the purpose is to discover.
func networkPluginsDiscovery(dynClient dynamic.Interface, clientSet kubernetes.Interface) (*ClusterNetwork, error) {
	osClusterNet, err := discoverOpenShift4Network(dynClient)
	if err != nil || osClusterNet != nil {
		return osClusterNet, err
	}

	weaveClusterNet, err := discoverWeaveNetwork(clientSet)
	if err != nil || weaveClusterNet != nil {
		return weaveClusterNet, err
	}

	canalClusterNet, err := discoverCanalFlannelNetwork(clientSet)
	if err != nil || canalClusterNet != nil {
		return canalClusterNet, err
	}

	ovnClusterNet, err := discoverOvnKubernetesNetwork(clientSet)
	if err != nil || ovnClusterNet != nil {
		return ovnClusterNet, err
	}

	calicoClusterNet, err := discoverCalicoNetwork(clientSet)
	if err != nil || calicoClusterNet != nil {
		return calicoClusterNet, err
	}

	return nil, nil
}

func getGlobalCIDRs(operatorClient client.Client, operatorNamespace string) (string, error) {
	if operatorClient == nil {
		return "", nil
	}

	existingCfg := submariner.Submariner{}

	err := operatorClient.Get(context.TODO(), types.NamespacedName{Namespace: operatorNamespace, Name: names.SubmarinerCrName}, &existingCfg)
	if err != nil {
		return "", errors.Wrap(err, "error retrieving Submariner resource")
	}

	globalCIDR := existingCfg.Spec.GlobalCIDR

	return globalCIDR, nil
}
