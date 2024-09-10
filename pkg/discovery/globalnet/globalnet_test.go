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

package globalnet_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/cidr"
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"
	"k8s.io/client-go/kubernetes/scheme"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const namespace = "test-ns"

var _ = Describe("AllocateAndUpdateGlobalCIDRConfigMap", func() {
	var client controllerClient.Client

	BeforeEach(func(ctx SpecContext) {
		client = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		Expect(globalnet.CreateConfigMap(ctx, client, true, "168.254.0.0/16",
			8192, namespace)).To(Succeed())
	})

	When("the globalnet CIDR is not specified", func() {
		const expGlobalCIDR = "168.254.0.0/19"

		It("should allocate a new one", func(ctx SpecContext) {
			netconfig := &globalnet.Config{
				ClusterID: "east",
			}

			Expect(globalnet.AllocateAndUpdateGlobalCIDRConfigMap(ctx, client, namespace,
				netconfig, reporter.Klog())).To(Succeed())
			Expect(netconfig.GlobalCIDR).To(Equal(expGlobalCIDR))

			globalnetInfo, _, err := globalnet.GetGlobalNetworks(ctx, client, namespace)
			Expect(err).To(Succeed())
			Expect(globalnetInfo.Clusters).To(HaveKeyWithValue(netconfig.ClusterID, &cidr.ClusterInfo{
				CIDRs:     []string{expGlobalCIDR},
				ClusterID: netconfig.ClusterID,
			}))

			netconfig.GlobalCIDR = ""
			Expect(globalnet.AllocateAndUpdateGlobalCIDRConfigMap(ctx, client, namespace,
				netconfig, reporter.Klog())).To(Succeed())
			Expect(netconfig.GlobalCIDR).To(Equal(expGlobalCIDR))
		})
	})

	When("the globalnet CIDR is specified", func() {
		const expGlobalCIDR = "168.254.0.0/15"

		It("should not allocate a new one", func(ctx SpecContext) {
			netconfig := &globalnet.Config{
				ClusterID:  "east",
				GlobalCIDR: expGlobalCIDR,
			}

			Expect(globalnet.AllocateAndUpdateGlobalCIDRConfigMap(ctx, client, namespace,
				netconfig, reporter.Klog())).To(Succeed())
			Expect(netconfig.GlobalCIDR).To(Equal(expGlobalCIDR))

			globalnetInfo, _, err := globalnet.GetGlobalNetworks(ctx, client, namespace)
			Expect(err).To(Succeed())
			Expect(globalnetInfo.Clusters).To(HaveKeyWithValue(netconfig.ClusterID, &cidr.ClusterInfo{
				CIDRs:     []string{expGlobalCIDR},
				ClusterID: netconfig.ClusterID,
			}))

			netconfig.GlobalCIDR = ""
			Expect(globalnet.AllocateAndUpdateGlobalCIDRConfigMap(ctx, client, namespace,
				netconfig, reporter.Klog())).To(Succeed())
			Expect(netconfig.GlobalCIDR).To(Equal(expGlobalCIDR))
		})
	})

	When("the globalnet cluster size is specified", func() {
		It("should allocate a CIDR", func(ctx SpecContext) {
			netconfig := &globalnet.Config{
				ClusterID:   "east",
				ClusterSize: 4096,
			}

			Expect(globalnet.AllocateAndUpdateGlobalCIDRConfigMap(ctx, client, namespace,
				netconfig, reporter.Klog())).To(Succeed())
			Expect(netconfig.GlobalCIDR).To(Equal("168.254.0.0/20"))
		})
	})
})

var _ = Describe("ValidateExistingGlobalNetworks", func() {
	var client controllerClient.Client

	BeforeEach(func() {
		client = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	})

	When("the existing globalnet config is valid", func() {
		It("should succeed", func(ctx SpecContext) {
			Expect(globalnet.CreateConfigMap(ctx, client, true, globalnet.DefaultGlobalnetCIDR,
				globalnet.DefaultGlobalnetClusterSize, namespace)).To(Succeed())

			Expect(globalnet.ValidateExistingGlobalNetworks(ctx, client, namespace)).To(Succeed())
		})
	})

	When("the globalnet config does not exist", func() {
		It("should succeed", func(ctx SpecContext) {
			Expect(globalnet.ValidateExistingGlobalNetworks(ctx, client, namespace)).To(Succeed())
		})
	})

	When("the existing globalnet CIDR is invalid", func() {
		It("should return an error", func(ctx SpecContext) {
			Expect(globalnet.CreateConfigMap(ctx, client, true, "169.254.0.0/16",
				globalnet.DefaultGlobalnetClusterSize, namespace)).To(Succeed())

			Expect(globalnet.ValidateExistingGlobalNetworks(ctx, client, namespace)).ToNot(Succeed())
		})
	})
})
