/*
© 2020 Red Hat, Inc. and others.

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

package globalnet

import (
	"fmt"
	"net"

	submarinerClientset "github.com/submariner-io/submariner/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GlobalNetwork struct {
	GlobalCIDRs  []string
	ServiceCIDRs []string
	ClusterId    string
}

func (gn *GlobalNetwork) Show() {
	if gn == nil {
		fmt.Println("    No global network details discovered")
	} else {
		fmt.Printf("    Discovered global network details for Cluster %s:\n", gn.ClusterId)
		fmt.Printf("        ServiceCidrs: %v\n", gn.ServiceCIDRs)
		fmt.Printf("        Global CIDRs: %v\n", gn.GlobalCIDRs)

	}
}

func ShowNetworks(networks map[string]*GlobalNetwork) {
	for _, network := range networks {
		network.Show()
	}
}

func Discover(client *submarinerClientset.Clientset, namespace string) (map[string]*GlobalNetwork, error) {
	clusters, err := client.SubmarinerV1().Clusters(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var globalNetworks = make(map[string]*GlobalNetwork)
	for _, cluster := range clusters.Items {
		globalNetwork := GlobalNetwork{
			GlobalCIDRs:  cluster.Spec.GlobalCIDR,
			ServiceCIDRs: cluster.Spec.ServiceCIDR,
			ClusterId:    cluster.Spec.ClusterID,
		}
		globalNetworks[cluster.Spec.ClusterID] = &globalNetwork
	}
	return globalNetworks, nil
}

func IsOverlappingCIDR(cidrList []string, cidr string) (bool, error) {
	_, newNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err
	}
	for _, v := range cidrList {
		_, baseNet, err := net.ParseCIDR(v)
		if err != nil {
			return false, err
		}
		if baseNet.Contains(newNet.IP) || newNet.Contains(baseNet.IP) {
			return true, nil
		}
	}
	return false, nil
}
