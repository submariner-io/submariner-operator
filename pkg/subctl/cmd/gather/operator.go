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
package gather

import (
	submarinerOp "github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func OperatorSubmariner(info Info, namespace string) {
	ResourcesToYAMLFile(info, schema.GroupVersionResource{
		Group:    submarinerOp.SchemeGroupVersion.Group,
		Version:  submarinerOp.SchemeGroupVersion.Version,
		Resource: "submariners",
	}, namespace, metav1.ListOptions{})
}

func OperatorServiceDiscovery(info Info, namespace string) {
	ResourcesToYAMLFile(info, schema.GroupVersionResource{
		Group:    submarinerOp.SchemeGroupVersion.Group,
		Version:  submarinerOp.SchemeGroupVersion.Version,
		Resource: "servicediscoveries",
	}, namespace, metav1.ListOptions{})
}

func GatewayDaemonSet(info Info, namespace string) {
	gatherDaemonSet(info, namespace, gatewayPodLabel)
}

func RouteAgentDaemonSet(info Info, namespace string) {
	gatherDaemonSet(info, namespace, routeagentPodLabel)
}

func GlobalnetDaemonSet(info Info, namespace string) {
	gatherDaemonSet(info, namespace, globalnetPodLabel)
}

func NetworkPluginSyncerDeployment(info Info, namespace string) {
	gatherDeployment(info, namespace, networkpluginSyncerPodLabel)
}

func LighthouseAgentDeployment(info Info, namespace string) {
	gatherDeployment(info, namespace, "app=submariner-lighthouse-agent")
}

func LighthouseCoreDNSDeployment(info Info, namespace string) {
	gatherDeployment(info, namespace, "app=submariner-lighthouse-coredns")
}
