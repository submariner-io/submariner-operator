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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testFlannelPodCIDR = "10.0.0.0/8"
)

var _ = Describe("Flannel Network", func() {
	When("the flannel DaemonSet and ConfigMap exist", func() {
		It("should return a ClusterNetwork with the plugin name and CIDRs set correctly", func() {
			clusterNet := testDiscoverFlannelWith(&flannelDaemonSet, &flannelCfgMap)
			Expect(clusterNet).NotTo(BeNil())
			Expect(clusterNet.NetworkPlugin).To(Equal(cni.Flannel))
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testFlannelPodCIDR}))
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDRFromService}))
		})
	})

	When("the flannel DaemonSet does not exist", func() {
		It("should return a ClusterNetwork with the generic plugin", func() {
			clusterNet := testDiscoverFlannelWith()
			Expect(clusterNet).NotTo(BeNil())
			Expect(clusterNet.NetworkPlugin).To(Equal(cni.Generic))
		})
	})

	When("the flannel ConfigMap does not exist", func() {
		It("should return a ClusterNetwork with the generic plugin", func() {
			clusterNet := testDiscoverFlannelWith(&flannelDaemonSet)
			Expect(clusterNet).NotTo(BeNil())
			Expect(clusterNet.NetworkPlugin).To(Equal(cni.Generic))
		})
	})
})

func testDiscoverFlannelWith(objects ...controllerClient.Object) *network.ClusterNetwork {
	client := newTestClient(objects...)
	clusterNet, err := network.Discover(client, "")
	Expect(err).NotTo(HaveOccurred())

	return clusterNet
}

var flannelDaemonSet = appsv1.DaemonSet{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "kube-flannel-ds",
		Namespace: metav1.NamespaceSystem,
	},
	Spec: appsv1.DaemonSetSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: corev1.PodSpec{
				Volumes: volumes,
			},
		},
	},
}

var volumes = []corev1.Volume{
	{
		Name: "flannel-cfg",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: "kube-flannel-cfg"},
			},
		},
	},
}

var flannelCfgMap = corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "kube-flannel-cfg",
		Namespace: metav1.NamespaceSystem,
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
