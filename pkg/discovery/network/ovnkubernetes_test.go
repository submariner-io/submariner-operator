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

package network_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner/pkg/cni"
	v1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const ovnKubeNamespace = "ovn-kubernetes"

var _ = Describe("OvnKubernetes Network", func() {
	const ovnKubeSvcTest = "ovnkube-db"

	When("ovn-kubernetes database is found, but no database service", func() {
		It("Should return error", func() {
			clusterNet, err := testOvnKubernetesDiscoveryWith(
				fakePodWithNamespace(ovnKubeNamespace, ovnKubeSvcTest, ovnKubeSvcTest, []string{}, []v1.EnvVar{}),
			)

			Expect(err).To(HaveOccurred())
			Expect(clusterNet).To(BeNil())
		})
	})

	When("ovn-kubernetes database and service found, no configmap", func() {
		It("Should return cluster network with default CIDRs", func() {
			clusterNet, err := testOvnKubernetesDiscoveryWith(
				fakePodWithNamespace(ovnKubeNamespace, ovnKubeSvcTest, ovnKubeSvcTest, []string{}, []v1.EnvVar{}),
				fakeService(ovnKubeNamespace, ovnKubeSvcTest, ovnKubeSvcTest),
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(clusterNet).NotTo(BeNil())
			Expect(clusterNet.NetworkPlugin).To(Equal(cni.OVNKubernetes))
			connectionStr := fmt.Sprintf("tcp:%s.%s", ovnKubeSvcTest, ovnKubeNamespace)
			Expect(clusterNet.PluginSettings["OVN_NBDB"]).To(Equal(connectionStr + ":6641"))
			Expect(clusterNet.PluginSettings["OVN_SBDB"]).To(Equal(connectionStr + ":6642"))
			Expect(clusterNet.PodCIDRs).To(HaveLen(0))
			Expect(clusterNet.ServiceCIDRs).To(HaveLen(1))
		})
	})

	When("ovn-kubernetes database, configmap and service found", func() {
		It("Should return cluster network with correct CIDRs", func() {
			clusterNet, err := testOvnKubernetesDiscoveryWith(
				fakePodWithNamespace(ovnKubeNamespace, ovnKubeSvcTest, ovnKubeSvcTest, []string{}, []v1.EnvVar{}),
				fakeService(ovnKubeNamespace, ovnKubeSvcTest, ovnKubeSvcTest),
				ovnFakeConfigMap(ovnKubeNamespace, "ovn-config"),
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(clusterNet).NotTo(BeNil())
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
		})
	})
})

func testOvnKubernetesDiscoveryWith(objects ...controllerClient.Object) (*network.ClusterNetwork, error) {
	client := newTestClient(objects...)
	return network.Discover(client, "")
}

func ovnFakeConfigMap(namespace, name string) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: v1meta.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data: map[string]string{
			"net_cidr": testPodCIDR,
			"svc_cidr": testServiceCIDR,
		},
	}
}
