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
	lhconstants "github.com/submariner-io/lighthouse/pkg/constants"
	discoveryv1beta1 "k8s.io/api/discovery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	mcsv1a1 "sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

const (
	lighthouseComponentsLabel = "component=submariner-lighthouse"
	coreDNSPodLabel           = "k8s-app=kube-dns"
)

func ServiceDiscoveryPodLogs(info Info) error {
	return gatherPodLogs(lighthouseComponentsLabel, info)
}

func CoreDNSPodLogs(info Info) error {
	return gatherPodLogs(coreDNSPodLabel, info)
}

func ServiceExports(info Info, namespace string) {
	ResourcesToYAMLFile(info, schema.GroupVersionResource{
		Group:    mcsv1a1.GroupName,
		Version:  mcsv1a1.GroupVersion.Version,
		Resource: "serviceexports",
	}, namespace, metav1.ListOptions{})
}

func ServiceImports(info Info, namespace string) {
	ResourcesToYAMLFile(info, schema.GroupVersionResource{
		Group:    mcsv1a1.GroupName,
		Version:  mcsv1a1.GroupVersion.Version,
		Resource: "serviceimports",
	}, namespace, metav1.ListOptions{})
}

func EndpointSlices(info Info, namespace string) {
	labelMap := map[string]string{
		discoveryv1beta1.LabelManagedBy: lhconstants.LabelValueManagedBy,
	}
	labelSelector := labels.Set(labelMap).String()

	ResourcesToYAMLFile(info, schema.GroupVersionResource{
		Group:    discoveryv1beta1.SchemeGroupVersion.Group,
		Version:  discoveryv1beta1.SchemeGroupVersion.Version,
		Resource: "endpointslices",
	}, namespace, metav1.ListOptions{LabelSelector: labelSelector})
}

func ConfigMapCoreDNS(info Info, namespace string) {
	fieldMap := map[string]string{
		"metadata.name": "coredns",
	}
	fieldSelector := fields.Set(fieldMap).String()
	gatherConfigMaps(info, namespace, metav1.ListOptions{FieldSelector: fieldSelector})
}

func ConfigMapLighthouseDNS(info Info, namespace string) {
	gatherConfigMaps(info, namespace, metav1.ListOptions{LabelSelector: lighthouseComponentsLabel})
}
