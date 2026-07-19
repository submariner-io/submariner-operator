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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner/pkg/cni"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Amazon VPC CNI Network", func() {
	awsNodeDaemonSet := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-node",
			Namespace: metav1.NamespaceSystem,
		},
	}

	awsNodePod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-node-xyz",
			Namespace: metav1.NamespaceSystem,
			Labels:    map[string]string{"k8s-app": "aws-node"},
		},
	}

	When("the aws-node DaemonSet exists", func() {
		It("should return a ClusterNetwork with the amazon-vpc-cni plugin", func(ctx SpecContext) {
			clusterNet := testDiscoverNetworkSuccess(ctx, &awsNodeDaemonSet, newServiceCIDR(testServiceCIDR),
				fakeKubeAPIServerPod(), fakeKubeControllerManagerPod())
			Expect(clusterNet.NetworkPlugin).To(Equal("amazon-vpc-cni"))
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
		})
	})

	When("only aws-node pods exist", func() {
		It("should return a ClusterNetwork with the amazon-vpc-cni plugin", func(ctx SpecContext) {
			clusterNet := testDiscoverNetworkSuccess(ctx, &awsNodePod, newServiceCIDR(testServiceCIDR))
			Expect(clusterNet.NetworkPlugin).To(Equal("amazon-vpc-cni"))
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
		})
	})

	When("aws-node is absent", func() {
		It("should fall back to the generic plugin", func(ctx SpecContext) {
			clusterNet := testDiscoverNetworkSuccess(ctx, newServiceCIDR(testServiceCIDR))
			Expect(clusterNet.NetworkPlugin).To(Equal(cni.Generic))
		})
	})
})
