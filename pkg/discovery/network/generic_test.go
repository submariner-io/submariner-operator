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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const testPodCIDR = "1.2.3.4/16"
const testServiceCIDR = "4.5.6.7/16"

var _ = Describe("discoverGenericNetwork", func() {
	When("There are no generic k8s pods to look at", func() {
		It("Should return nil cluster network", func() {
			clusterNet := testDiscoverGenericWith()
			Expect(clusterNet).To(BeNil())
		})
	})

	When("There is a kube-proxy with no expected parameters", func() {
		It("Should return nil cluster network", func() {
			clusterNet := testDiscoverGenericWith(
				fakePod("kube-proxy", []string{"kube-proxy", "--cluster-ABCD=1.2.3.4"}, []v1.EnvVar{}),
			)
			Expect(clusterNet).To(BeNil())
		})
	})

	When("There is a kube-controller with no expected parameters", func() {
		It("Should return nil cluster network", func() {
			clusterNet := testDiscoverGenericWith(
				fakePod("kube-controller", []string{"kube-controller", "--cluster-ABCD=1.2.3.4"}, []v1.EnvVar{}),
			)
			Expect(clusterNet).To(BeNil())
		})
	})

	When("There is a kube-api with no expected parameters", func() {
		It("Should return nil cluster network", func() {
			clusterNet := testDiscoverGenericWith(
				fakePod("kube-controller", []string{"kube-api", "--cluster-ABCD=1.2.3.4"}, []v1.EnvVar{}),
			)
			Expect(clusterNet).To(BeNil())
		})
	})

	When("There is a kube-controller pod with the right parameter", func() {

		var clusterNet *ClusterNetwork

		BeforeEach(func() {
			clusterNet = testDiscoverGenericWith(
				fakePod("kube-controller-manager", []string{"kube-controller-manager", "--cluster-cidr=" + testPodCIDR}, []v1.EnvVar{}),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("Should return the ClusterNetwork structure with PodCIDR", func() {
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
		})

		It("Should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo("generic"))
		})

		It("Should return the ClusterNetwork structure with empty service CIDR", func() {
			Expect(clusterNet.ServiceCIDRs).To(BeEmpty())
		})
	})

	When("There is a kube-proxy pod but no kube-controller", func() {

		var clusterNet *ClusterNetwork

		BeforeEach(func() {
			clusterNet = testDiscoverGenericWith(
				fakePod("kube-proxy", []string{"kube-proxy", "--cluster-cidr=" + testPodCIDR}, []v1.EnvVar{}),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("Should return the ClusterNetwork structure with PodCIDR", func() {
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
		})

		It("Should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo("generic"))
		})

		It("Should return the ClusterNetwork structure with empty service CIDR", func() {
			Expect(clusterNet.ServiceCIDRs).To(BeEmpty())
		})

	})

	When("There is a kubeapi pod", func() {
		var clusterNet *ClusterNetwork

		BeforeEach(func() {
			clusterNet = testDiscoverGenericWith(
				fakePod("kube-apiserver", []string{"kube-apiserver", "--service-cluster-ip-range=" + testServiceCIDR}, []v1.EnvVar{}),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("Should return the ClusterNetwork structure with ServiceCIDRs", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
		})

		It("Should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo("generic"))
		})

		It("Should return the ClusterNetwork structure with empty PodCIDRs", func() {
			Expect(clusterNet.PodCIDRs).To(BeEmpty())
		})

	})

	When("There is a kube-proxy and api pods", func() {

		var clusterNet *ClusterNetwork

		BeforeEach(func() {
			clusterNet = testDiscoverGenericWith(
				fakePod("kube-proxy", []string{"kube-proxy", "--cluster-cidr=" + testPodCIDR}, []v1.EnvVar{}),
				fakePod("kube-apiserver", []string{"kube-apiserver", "--service-cluster-ip-range=" + testServiceCIDR}, []v1.EnvVar{}),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("Should return ClusterNetwork with all CIDRs", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
		})

		It("Should identify the network plugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo("generic"))
		})
	})
})

func testDiscoverGenericWith(objects ...runtime.Object) *ClusterNetwork {
	clientSet := fake.NewSimpleClientset(objects...)
	clusterNet, err := discoverGenericNetwork(clientSet)
	Expect(err).NotTo(HaveOccurred())
	return clusterNet
}
