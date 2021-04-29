/*
© 2021 Red Hat, Inc. and others.

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
	v1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("discoverCalicoNetwork", func() {
	When("There are no generic k8s pods to look at", func() {
		It("Should return the ClusterNetwork structure with the pod CIDR and the service CIDR", func() {
			clusterNet := testDiscoverCalicoWith(&calicoCfgMap)
			Expect(clusterNet).NotTo(BeNil())
			Expect(clusterNet.NetworkPlugin).To(Equal("calico"))
			Expect(clusterNet.PodCIDRs).To(BeEmpty())
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDRFromService}))
		})
	})

	When("There is a kube-api pod", func() {
		It("Should return the ClusterNetwork structure with the pod CIDR and the service CIDR", func() {
			clusterNet := testDiscoverWith(
				&calicoCfgMap,
				fakePod("kube-apiserver", []string{"kube-apiserver", "--service-cluster-ip-range=" + testServiceCIDR}, []v1.EnvVar{}),
				fakePod("kube-controller-manager", []string{"kube-controller-manager", "--cluster-cidr=" + testPodCIDR}, []v1.EnvVar{}),
			)
			Expect(clusterNet).NotTo(BeNil())
			Expect(clusterNet.NetworkPlugin).To(Equal("calico"))
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
		})
	})
})

func testDiscoverCalicoWith(objects ...runtime.Object) *ClusterNetwork {
	clientSet := newTestClient(objects...)
	clusterNet, err := discoverCalicoNetwork(clientSet)
	Expect(err).NotTo(HaveOccurred())
	return clusterNet
}

var calicoCfgMap v1.ConfigMap = v1.ConfigMap{
	ObjectMeta: v1meta.ObjectMeta{
		Name:      "calico-config",
		Namespace: "kube-system",
	},
	Data: map[string]string{},
}
