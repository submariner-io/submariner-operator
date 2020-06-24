/*
© 2019 Red Hat, Inc. and others.

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
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
)

type ClusterNetwork struct {
	PodCIDRs      []string
	ServiceCIDRs  []string
	NetworkPlugin string
	GlobalCIDR    string
}

func (cn *ClusterNetwork) Show() {
	if cn == nil {
		fmt.Println("    No network details discovered")
	} else {
		fmt.Printf("    Discovered network details:\n")
		fmt.Printf("        Network plugin:  %s\n", cn.NetworkPlugin)
		fmt.Printf("        Service CIDRs:   %v\n", cn.ServiceCIDRs)
		fmt.Printf("        Cluster CIDRs:   %v\n", cn.PodCIDRs)
		if cn.GlobalCIDR != "" {
			fmt.Printf("        Global CIDR:     %v\n", cn.GlobalCIDR)
		}
	}
}

func (cn *ClusterNetwork) IsComplete() bool {
	return cn != nil && len(cn.ServiceCIDRs) > 0 && len(cn.PodCIDRs) > 0
}

func Discover(dynClient dynamic.Interface, clientSet kubernetes.Interface, submClient submarinerclientset.Interface, operatorNamespace string) (*ClusterNetwork, error) {
	discovery, err := networkPluginsDiscovery(dynClient, clientSet)
	globalCIDR, _ := getGlobalCIDRs(submClient, operatorNamespace)
	discovery.GlobalCIDR = globalCIDR

	if err == nil && discovery != nil {
		if discovery.IsComplete() {
			return discovery, nil
		} else {
			// If the info we got from the non-generic plugins is incomplete
			// try to complete with the generic discovery mechanisms
			genericNet, err := discoverGenericNetwork(clientSet)
			if genericNet == nil || err != nil {
				return discovery, nil
			}

			if len(discovery.ServiceCIDRs) == 0 {
				discovery.ServiceCIDRs = genericNet.ServiceCIDRs
			}
			if len(discovery.PodCIDRs) == 0 {
				discovery.PodCIDRs = genericNet.PodCIDRs
			}
			return discovery, nil
		}
	} else {
		// If nothing specific was discovered, use the generic discovery
		genericNet, err := discoverGenericNetwork(clientSet)
		if err == nil && genericNet != nil {
			return genericNet, nil
		}
		return nil, err
	}
}

func networkPluginsDiscovery(dynClient dynamic.Interface, clientSet kubernetes.Interface) (*ClusterNetwork, error) {

	osClusterNet, err := discoverOpenShift4Network(dynClient)
	if err == nil && osClusterNet != nil {
		return osClusterNet, nil
	}

	weaveClusterNet, err := discoverWeaveNetwork(clientSet)
	if err == nil && weaveClusterNet != nil {
		return weaveClusterNet, nil
	}

	canalClusterNet, err := discoverCanalFlannelNetwork(clientSet)
	if err == nil && canalClusterNet != nil {
		return canalClusterNet, nil
	}

	return nil, nil
}

func getGlobalCIDRs(submClient submarinerclientset.Interface, operatorNamespace string) (string, error) {
	if submClient == nil {
		return "", nil
	}
	existingCfg, err := submClient.SubmarinerV1alpha1().Submariners(operatorNamespace).Get(submarinercr.SubmarinerName, v1.GetOptions{})
	if err != nil {
		return "", err
	}

	globalCIDR := existingCfg.Spec.GlobalCIDR
	return globalCIDR, nil
}
