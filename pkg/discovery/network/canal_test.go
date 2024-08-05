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

var _ = Describe("Canal Flannel Network", func() {
	canalDaemonSet := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "canal",
			Namespace: metav1.NamespaceSystem,
			Labels:    map[string]string{"k8s-app": "canal"},
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{flannelCfgVolume},
				},
			},
		},
	}

	When("the canal DaemonSet and ConfigMap exist", func() {
		It("should return a ClusterNetwork with the plugin name and CIDRs set correctly", func(ctx SpecContext) {
			clusterNet := testDiscoverNetworkSuccess(ctx, &flannelCfgMap, &canalDaemonSet)
			Expect(clusterNet).NotTo(BeNil())
			Expect(clusterNet.NetworkPlugin).To(Equal(cni.CanalFlannel))
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testFlannelPodCIDR}))
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDRFromService}))
		})
	})

	When("a K8s API server pod exists", func() {
		It("should return a ClusterNetwork with the plugin name and CIDRs set correctly", func(ctx SpecContext) {
			clusterNet := testDiscoverNetworkSuccess(ctx, &flannelCfgMap, &canalDaemonSet, fakeKubeAPIServerPod())
			Expect(clusterNet).NotTo(BeNil())
			Expect(clusterNet.NetworkPlugin).To(Equal(cni.CanalFlannel))
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testFlannelPodCIDR}))
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
		})
	})

	When("the flannel DaemonSet does not exist", func() {
		It("should return a ClusterNetwork with the generic plugin", func(ctx SpecContext) {
			clusterNet := testDiscoverNetworkSuccess(ctx)
			Expect(clusterNet).NotTo(BeNil())
			Expect(clusterNet.NetworkPlugin).To(Equal(cni.Generic))
		})
	})

	When("the flannel ConfigMap does not exist", func() {
		It("should return a ClusterNetwork with the generic plugin", func(ctx SpecContext) {
			clusterNet := testDiscoverNetworkSuccess(ctx, &canalDaemonSet)
			Expect(clusterNet).NotTo(BeNil())
			Expect(clusterNet.NetworkPlugin).To(Equal(cni.Generic))
		})
	})
})
