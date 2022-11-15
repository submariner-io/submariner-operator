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
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/fake"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner/pkg/cni"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testPodCIDR                = "1.2.3.4/16"
	testServiceCIDR            = "4.5.6.7/16"
	testServiceCIDRFromService = "7.8.9.10/16"
)

var _ = Describe("Generic Network", func() {
	When("There is a kube-proxy with no expected parameters", func() {
		var clusterNet *network.ClusterNetwork

		BeforeEach(func() {
			clusterNet = testDiscoverGenericWith(
				fakePod("kube-proxy", []string{"kube-proxy", "--cluster-ABCD=1.2.3.4"}, []corev1.EnvVar{}),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("Should return the ClusterNetwork structure with empty PodCIDRs", func() {
			Expect(clusterNet.PodCIDRs).To(BeEmpty())
		})

		It("Should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})

		It("Should return the ClusterNetwork structure with the service CIDR", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDRFromService}))
		})
	})

	When("There is a kube-controller with no expected parameters", func() {
		var clusterNet *network.ClusterNetwork

		BeforeEach(func() {
			clusterNet = testDiscoverGenericWith(
				fakePod("kube-controller-manager", []string{"kube-controller-manager", "--cluster-ABCD=1.2.3.4"}, []corev1.EnvVar{}),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("Should return the ClusterNetwork structure with empty PodCIDRs", func() {
			Expect(clusterNet.PodCIDRs).To(BeEmpty())
		})

		It("Should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})

		It("Should return the ClusterNetwork structure with the service CIDR", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDRFromService}))
		})
	})

	When("There is a kube-api with no expected parameters", func() {
		var clusterNet *network.ClusterNetwork

		BeforeEach(func() {
			clusterNet = testDiscoverGenericWith(
				fakePod("kube-apiserver", []string{"kube-apiserver", "--cluster-ABCD=1.2.3.4"}, []corev1.EnvVar{}),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("Should return the ClusterNetwork structure with empty PodCIDRs", func() {
			Expect(clusterNet.PodCIDRs).To(BeEmpty())
		})

		It("Should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})

		It("Should return the ClusterNetwork structure with the service CIDR", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDRFromService}))
		})
	})

	When("There is a kube-controller pod with the right parameter", func() {
		var clusterNet *network.ClusterNetwork

		BeforeEach(func() {
			clusterNet = testDiscoverGenericWith(
				fakePod("kube-controller-manager", []string{"kube-controller-manager", "--cluster-cidr=" + testPodCIDR}, []corev1.EnvVar{}),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("Should return the ClusterNetwork structure with pod CIDR", func() {
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
		})

		It("Should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})

		It("Should return the ClusterNetwork structure with the service CIDR", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDRFromService}))
		})
	})

	When("There is a kube-proxy pod but no kube-controller", func() {
		var clusterNet *network.ClusterNetwork

		BeforeEach(func() {
			clusterNet = testDiscoverGenericWith(
				fakePod("kube-proxy", []string{"kube-proxy", "--cluster-cidr=" + testPodCIDR}, []corev1.EnvVar{}),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("Should return the ClusterNetwork structure with PodCIDR", func() {
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
		})

		It("Should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})

		It("Should return the ClusterNetwork structure with the service CIDR", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDRFromService}))
		})
	})

	When("There is a kubeapi pod", func() {
		var clusterNet *network.ClusterNetwork

		BeforeEach(func() {
			clusterNet = testDiscoverGenericWith(
				fakePod("kube-apiserver", []string{"kube-apiserver", "--service-cluster-ip-range=" + testServiceCIDR}, []corev1.EnvVar{}),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("Should return the ClusterNetwork structure with ServiceCIDRs", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
		})

		It("Should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})

		It("Should return the ClusterNetwork structure with empty PodCIDRs", func() {
			Expect(clusterNet.PodCIDRs).To(BeEmpty())
		})
	})

	When("There is a kube-proxy and api pods", func() {
		var clusterNet *network.ClusterNetwork

		BeforeEach(func() {
			clusterNet = testDiscoverGenericWith(
				fakePod("kube-proxy", []string{"kube-proxy", "--cluster-cidr=" + testPodCIDR}, []corev1.EnvVar{}),
				fakePod("kube-apiserver", []string{"kube-apiserver", "--service-cluster-ip-range=" + testServiceCIDR}, []corev1.EnvVar{}),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("Should return ClusterNetwork with all CIDRs", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
		})

		It("Should identify the network plugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})
	})

	When("No pod CIDR information exists on any node", func() {
		var clusterNet *network.ClusterNetwork

		BeforeEach(func() {
			clusterNet = testDiscoverGenericWith(
				fakeNode("node1", ""),
				fakeNode("node2", ""),
			)
		})

		It("Should return the ClusterNetwork structure with empty PodCIDRs", func() {
			Expect(clusterNet.PodCIDRs).To(BeEmpty())
		})

		It("Should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})

		It("Should return the ClusterNetwork structure with the service CIDR", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDRFromService}))
		})
	})

	When("Pod CIDR information exists on a node", func() {
		var clusterNet *network.ClusterNetwork

		BeforeEach(func() {
			clusterNet = testDiscoverGenericWith(
				fakeNode("node1", ""),
				fakeNode("node2", testPodCIDR),
			)
		})

		It("Should return the ClusterNetwork structure with the pod CIDR", func() {
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
		})

		It("Should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})

		It("Should return the ClusterNetwork structure with the service CIDR", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDRFromService}))
		})
	})

	When("Both pod and service CIDR information exists", func() {
		var clusterNet *network.ClusterNetwork

		BeforeEach(func() {
			clusterNet = testDiscoverGenericWith(
				fakeNode("node1", testPodCIDR),
				fakePod("kube-apiserver", []string{"kube-apiserver", "--service-cluster-ip-range=" + testServiceCIDR}, []corev1.EnvVar{}),
			)
		})

		It("Should return ClusterNetwork with all CIDRs", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
		})

		It("Should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})
	})

	When("No kube-api pod exists and invalid service creation returns no error", func() {
		It("Should return error and nil cluster network", func() {
			client := fakeClient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
			clusterNet, err := network.Discover(context.TODO(), client, "")
			Expect(err).To(HaveOccurred())
			Expect(clusterNet).To(BeNil())
		})
	})

	When("No kube-api pod exists and invalid service creation returns an unexpected error", func() {
		It("Should return error and nil cluster network", func() {
			// Inject error for create services to return expectedErr
			client := fake.NewReactingClient(nil).AddReactor(fake.Create, &corev1.Service{},
				fake.FailingReaction(fmt.Errorf("%s", testServiceCIDR)))

			clusterNet, err := network.Discover(context.TODO(), client, "")
			Expect(err).To(HaveOccurred())
			Expect(clusterNet).To(BeNil())
		})
	})

	When("No kube-api pod exists and invalid service creation returns the expected error", func() {
		var clusterNet *network.ClusterNetwork

		BeforeEach(func() {
			clusterNet = testDiscoverGenericWith()
		})

		It("Should return the ClusterNetwork structure with empty pod CIDR", func() {
			Expect(clusterNet.PodCIDRs).To(BeEmpty())
		})

		It("Should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})

		It("Should return the ClusterNetwork structure with the service CIDR", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDRFromService}))
		})
	})
})

func testDiscoverGenericWith(objects ...controllerClient.Object) *network.ClusterNetwork {
	client := newTestClient(objects...)
	clusterNet, err := network.Discover(context.TODO(), client, "")
	Expect(err).NotTo(HaveOccurred())

	return clusterNet
}

func newTestClient(objects ...controllerClient.Object) controllerClient.Client {
	// Inject error for create services to return expectedErr
	return fake.NewReactingClient(fakeClient.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()).
		AddReactor(fake.Create, &corev1.Service{}, fake.FailingReaction(fmt.Errorf("The Service \"invalid-svc\" is invalid: "+
			"spec.clusterIPs: Invalid value: []string{\"1.1.1.1\"}: failed to "+
			"allocated ip:1.1.1.1 with error:provided IP is not in the valid range. "+
			"The range of valid IPs is %s", testServiceCIDRFromService)))
}
