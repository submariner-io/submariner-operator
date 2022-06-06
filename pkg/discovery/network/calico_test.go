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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner/pkg/cni"
	v1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("Calico Network", func() {
	var (
		initObjs     []runtime.Object
		clusterNet   *network.ClusterNetwork
		err          error
		calicoCfgMap = &v1.ConfigMap{
			ObjectMeta: v1meta.ObjectMeta{
				Name:      "calico-config",
				Namespace: "kube-system",
			},
			Data: map[string]string{},
		}
	)

	BeforeEach(func() {
		initObjs = nil
	})

	JustBeforeEach(func() {
		clientSet := newTestClient(initObjs...)
		clusterNet, err = network.Discover(nil, clientSet, nil, "")
	})

	When("no kube pod information is available", func() {
		JustBeforeEach(func() {
			initObjs = []runtime.Object{
				calicoCfgMap,
			}

			clientSet := newTestClient(initObjs...)
			clusterNet, err = network.Discover(nil, clientSet, nil, "")
		})

		It("should return a ClusterNetwork with only service CIDRs", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(clusterNet).NotTo(BeNil())
			Expect(clusterNet.NetworkPlugin).To(Equal(cni.Calico))
			Expect(clusterNet.PodCIDRs).To(BeEmpty())
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDRFromService}))
		})
	})

	When("kube pod information is available", func() {
		JustBeforeEach(func() {
			initObjs = []runtime.Object{
				calicoCfgMap,
				fakePod("kube-apiserver", []string{"kube-apiserver", "--service-cluster-ip-range=" + testServiceCIDR}, []v1.EnvVar{}),
				fakePod("kube-controller-manager", []string{"kube-controller-manager", "--cluster-cidr=" + testPodCIDR}, []v1.EnvVar{}),
			}

			clientSet := newTestClient(initObjs...)
			clusterNet, err = network.Discover(nil, clientSet, nil, "")
		})
		It("should return a ClusterNetwork with pod and service CIDRs", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(clusterNet).NotTo(BeNil())
			Expect(clusterNet.NetworkPlugin).To(Equal(cni.Calico))
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
		})
	})
})
