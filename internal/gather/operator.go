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

package gather

import (
	submarinerOp "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func gatherSubmariners(info *Info, namespace string) {
	ResourcesToYAMLFile(info, schema.GroupVersionResource{
		Group:    submarinerOp.SchemeGroupVersion.Group,
		Version:  submarinerOp.SchemeGroupVersion.Version,
		Resource: "submariners",
	}, namespace, metav1.ListOptions{})
}

func gatherServiceDiscoveries(info *Info, namespace string) {
	ResourcesToYAMLFile(info, schema.GroupVersionResource{
		Group:    submarinerOp.SchemeGroupVersion.Group,
		Version:  submarinerOp.SchemeGroupVersion.Version,
		Resource: "servicediscoveries",
	}, namespace, metav1.ListOptions{})
}

func gatherSubmarinerOperatorDeployment(info *Info, namespace string) {
	gatherDeployment(info, namespace, metav1.ListOptions{FieldSelector: fields.Set(map[string]string{
		"metadata.name": "submariner-operator",
	}).String()})
}

func gatherGatewayDaemonSet(info *Info, namespace string) {
	gatherDaemonSet(info, namespace, metav1.ListOptions{LabelSelector: gatewayPodLabel})
}

func gatherRouteAgentDaemonSet(info *Info, namespace string) {
	gatherDaemonSet(info, namespace, metav1.ListOptions{LabelSelector: routeagentPodLabel})
}

func gatherGlobalnetDaemonSet(info *Info, namespace string) {
	gatherDaemonSet(info, namespace, metav1.ListOptions{LabelSelector: globalnetPodLabel})
}

func gatherNetworkPluginSyncerDeployment(info *Info, namespace string) {
	gatherDeployment(info, namespace, metav1.ListOptions{LabelSelector: networkpluginSyncerPodLabel})
}

func gatherLighthouseAgentDeployment(info *Info, namespace string) {
	gatherDeployment(info, namespace, metav1.ListOptions{LabelSelector: "app=submariner-lighthouse-agent"})
}

func gatherLighthouseCoreDNSDeployment(info *Info, namespace string) {
	gatherDeployment(info, namespace, metav1.ListOptions{LabelSelector: "app=submariner-lighthouse-coredns"})
}

func gatherSubmarinerOperatorPodLogs(info *Info) {
	gatherPodLogs("name=submariner-operator", info)
}
