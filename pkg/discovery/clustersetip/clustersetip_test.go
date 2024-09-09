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

package clustersetip_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/cidr"
	"github.com/submariner-io/submariner-operator/pkg/discovery/clustersetip"
	"k8s.io/client-go/kubernetes/scheme"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const namespace = "test-ns"

var _ = Describe("AllocateCIDRFromConfigMap", func() {
	var client controllerClient.Client

	BeforeEach(func(ctx SpecContext) {
		client = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		Expect(clustersetip.CreateConfigMap(ctx, client, true, "168.254.0.0/16",
			8192, namespace)).To(Succeed())
	})

	When("the clustersetip CIDR is not specified", func() {
		const expClustersetIPCIDR = "168.254.0.0/20"

		It("should allocate a new one", func(ctx SpecContext) {
			netconfig := &clustersetip.Config{
				ClusterID: "east",
			}

			Expect(clustersetip.AllocateCIDRFromConfigMap(ctx, client, namespace,
				netconfig, reporter.Klog())).To(Succeed())
			Expect(netconfig.ClustersetIPCIDR).To(Equal(expClustersetIPCIDR))

			clustersetipInfo, _, err := clustersetip.GetClustersetIPNetworks(ctx, client, namespace)
			Expect(err).To(Succeed())
			Expect(clustersetipInfo.Clusters).To(HaveKeyWithValue(netconfig.ClusterID, &cidr.ClusterInfo{
				CIDRs:     []string{expClustersetIPCIDR},
				ClusterID: netconfig.ClusterID,
			}))

			netconfig.ClustersetIPCIDR = ""
			Expect(clustersetip.AllocateCIDRFromConfigMap(ctx, client, namespace,
				netconfig, reporter.Klog())).To(Succeed())
			Expect(netconfig.ClustersetIPCIDR).To(Equal(expClustersetIPCIDR))
		})
	})

	When("the clustersetip CIDR is specified", func() {
		const expClustersetIPCIDR = "168.254.0.0/15"

		It("should not allocate a new one", func(ctx SpecContext) {
			netconfig := &clustersetip.Config{
				ClusterID:        "east",
				ClustersetIPCIDR: expClustersetIPCIDR,
			}

			Expect(clustersetip.AllocateCIDRFromConfigMap(ctx, client, namespace,
				netconfig, reporter.Klog())).To(Succeed())
			Expect(netconfig.ClustersetIPCIDR).To(Equal(expClustersetIPCIDR))

			clustersetipInfo, _, err := clustersetip.GetClustersetIPNetworks(ctx, client, namespace)
			Expect(err).To(Succeed())
			Expect(clustersetipInfo.Clusters).To(HaveKeyWithValue(netconfig.ClusterID, &cidr.ClusterInfo{
				CIDRs:     []string{expClustersetIPCIDR},
				ClusterID: netconfig.ClusterID,
			}))

			netconfig.ClustersetIPCIDR = ""
			Expect(clustersetip.AllocateCIDRFromConfigMap(ctx, client, namespace,
				netconfig, reporter.Klog())).To(Succeed())
			Expect(netconfig.ClustersetIPCIDR).To(Equal(expClustersetIPCIDR))
		})
	})

	When("the clustersetip cluster size is specified", func() {
		It("should allocate a CIDR", func(ctx SpecContext) {
			netconfig := &clustersetip.Config{
				ClusterID:      "east",
				AllocationSize: 1024,
			}

			Expect(clustersetip.AllocateCIDRFromConfigMap(ctx, client, namespace,
				netconfig, reporter.Klog())).To(Succeed())
			Expect(netconfig.ClustersetIPCIDR).To(Equal("168.254.0.0/22"))
		})
	})
})

var _ = Describe("ValidateExistingClustersetIPNetworks", func() {
	var client controllerClient.Client

	BeforeEach(func() {
		client = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	})

	When("the existing clustersetip config is valid", func() {
		It("should succeed", func(ctx SpecContext) {
			Expect(clustersetip.CreateConfigMap(ctx, client, true, clustersetip.DefaultCIDR,
				clustersetip.DefaultAllocationSize, namespace)).To(Succeed())

			Expect(clustersetip.ValidateExistingClustersetIPNetworks(ctx, client, namespace)).To(Succeed())
		})
	})

	When("the clustersetip config does not exist", func() {
		It("should succeed", func(ctx SpecContext) {
			Expect(clustersetip.ValidateExistingClustersetIPNetworks(ctx, client, namespace)).To(Succeed())
		})
	})

	When("the existing clustersetip CIDR is invalid", func() {
		It("should return an error", func(ctx SpecContext) {
			Expect(clustersetip.CreateConfigMap(ctx, client, true, "169.254.0.0/16",
				clustersetip.DefaultAllocationSize, namespace)).To(Succeed())

			Expect(clustersetip.ValidateExistingClustersetIPNetworks(ctx, client, namespace)).ToNot(Succeed())
		})
	})
})
