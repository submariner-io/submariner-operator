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
	"github.com/submariner-io/submariner/pkg/routeagent_driver/constants"
	v1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("discoverCanalFlannelNetwork", func() {
	When("There are no generic k8s pods to look at", func() {
		It("Should return the ClusterNetwork structure with the pod CIDR and the service CIDR", func() {
			clusterNet := testDiscoverCanalFlannelWith(&canalFlannelCfgMap)
			Expect(clusterNet).NotTo(BeNil())
			Expect(clusterNet.NetworkPlugin).To(Equal(constants.NetworkPluginCanalFlannel))
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testCannalFlannelPodCIDR}))
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDRFromService}))
		})
	})

	When("There is a kube-api pod", func() {

		It("Should return the ClusterNetwork structure with the pod CIDR and the service CIDR", func() {
			clusterNet := testDiscoverWith(
				&canalFlannelCfgMap,
				fakePod("kube-apiserver", []string{"kube-apiserver", "--service-cluster-ip-range=" + testServiceCIDR}, []v1.EnvVar{}),
			)
			Expect(clusterNet).NotTo(BeNil())
			Expect(clusterNet.NetworkPlugin).To(Equal(constants.NetworkPluginCanalFlannel))
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testCannalFlannelPodCIDR}))
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
		})

	})
})

func testDiscoverCanalFlannelWith(objects ...runtime.Object) *ClusterNetwork {
	clientSet := newTestClient(objects...)
	clusterNet, err := discoverCanalFlannelNetwork(clientSet)
	Expect(err).NotTo(HaveOccurred())
	return clusterNet
}

func testDiscoverWith(objects ...runtime.Object) *ClusterNetwork {
	clientSet := newTestClient(objects...)
	clusterNet, err := Discover(nil, clientSet, nil, "")
	Expect(err).NotTo(HaveOccurred())
	return clusterNet
}

var canalFlannelCfgMap v1.ConfigMap = v1.ConfigMap{
	ObjectMeta: v1meta.ObjectMeta{
		Name:      "canal-config",
		Namespace: "kube-system",
	},
	Data: map[string]string{
		"net-conf.json": `{
			"Network": "10.0.0.0/8",
			"SubnetLen": 20,
			"SubnetMin": "10.10.0.0",
			"SubnetMax": "10.99.0.0",
			"Backend": {
				"Type": "udp",
				"Port": 7890
			}
		}`,
	},
}

const testCannalFlannelPodCIDR = "10.0.0.0/8"
