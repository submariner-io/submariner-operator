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
	submarinerv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	gatewayPodLabel    = "app=submariner-gateway"
	routeagentPodLabel = "app=submariner-routeagent"
)

func GatewayPodLogs(info *Info) error {
	return gatherPodLogs(gatewayPodLabel, info)
}

func RouteAgentPodLogs(info *Info) error {
	return gatherPodLogs(routeagentPodLabel, info)
}

func Endpoints(info *Info, namespace string) {
	ResourcesToYAMLFile(info, schema.GroupVersionResource{
		Group:    submarinerv1.SchemeGroupVersion.Group,
		Version:  submarinerv1.SchemeGroupVersion.Version,
		Resource: "endpoints",
	}, namespace)
}

func Clusters(info *Info, namespace string) {
	ResourcesToYAMLFile(info, schema.GroupVersionResource{
		Group:    submarinerv1.SchemeGroupVersion.Group,
		Version:  submarinerv1.SchemeGroupVersion.Version,
		Resource: "clusters",
	}, namespace)
}

func Gateways(info *Info, namespace string) {
	ResourcesToYAMLFile(info, schema.GroupVersionResource{
		Group:    submarinerv1.SchemeGroupVersion.Group,
		Version:  submarinerv1.SchemeGroupVersion.Version,
		Resource: "gateways",
	}, namespace)
}
