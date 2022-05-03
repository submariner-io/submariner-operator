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
	lhconstants "github.com/submariner-io/lighthouse/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	mcsv1a1 "sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

const (
	lighthouseComponentsLabel = "component=submariner-lighthouse"
	k8sCoreDNSPodLabel        = "k8s-app=kube-dns"
	ocpCoreDNSPodLabel        = "dns.operator.openshift.io/daemonset-dns=default"
	internalSvcLabel          = "submariner.io/exportedServiceRef"
)

func gatherServiceDiscoveryPodLogs(info *Info) {
	gatherPodLogs(lighthouseComponentsLabel, info)
}

func gatherCoreDNSPodLogs(info *Info) {
	if isCoreDNSTypeOcp(info) {
		gatherPodLogsByContainer(ocpCoreDNSPodLabel, "dns", info)
	} else {
		gatherPodLogs(k8sCoreDNSPodLabel, info)
	}
}

func gatherServiceExports(info *Info, namespace string) {
	ResourcesToYAMLFile(info, schema.GroupVersionResource{
		Group:    mcsv1a1.GroupName,
		Version:  mcsv1a1.GroupVersion.Version,
		Resource: "serviceexports",
	}, namespace, metav1.ListOptions{})
}

func gatherServiceImports(info *Info, namespace string) {
	ResourcesToYAMLFile(info, schema.GroupVersionResource{
		Group:    mcsv1a1.GroupName,
		Version:  mcsv1a1.GroupVersion.Version,
		Resource: "serviceimports",
	}, namespace, metav1.ListOptions{})
}

func gatherEndpointSlices(info *Info, namespace string) {
	labelMap := map[string]string{
		discoveryv1.LabelManagedBy: lhconstants.LabelValueManagedBy,
	}
	labelSelector := labels.Set(labelMap).String()

	ResourcesToYAMLFile(info, schema.GroupVersionResource{
		Group:    discoveryv1.SchemeGroupVersion.Group,
		Version:  discoveryv1.SchemeGroupVersion.Version,
		Resource: "endpointslices",
	}, namespace, metav1.ListOptions{LabelSelector: labelSelector})
}

func gatherConfigMapCoreDNS(info *Info) {
	namespace := "kube-system"
	name := "coredns"

	if isCoreDNSTypeOcp(info) {
		namespace = "openshift-dns"
		name = "dns-default"
	}

	fieldMap := map[string]string{
		"metadata.name": name,
	}

	fieldSelector := fields.Set(fieldMap).String()

	gatherConfigMaps(info, namespace, metav1.ListOptions{FieldSelector: fieldSelector})

	// Gather custom configname for AKS type deployments
	if info.ServiceDiscovery.Spec.CoreDNSCustomConfig != nil {
		name = info.ServiceDiscovery.Spec.CoreDNSCustomConfig.ConfigMapName

		if info.ServiceDiscovery.Spec.CoreDNSCustomConfig.Namespace != "" {
			namespace = info.ServiceDiscovery.Spec.CoreDNSCustomConfig.Namespace
		}

		fieldMap := map[string]string{
			"metadata.name": name,
		}

		fieldSelector := fields.Set(fieldMap).String()

		gatherConfigMaps(info, namespace, metav1.ListOptions{FieldSelector: fieldSelector})
	}
}

// gatherLabeledServices gathers a service based on the label provided.
func gatherLabeledServices(info *Info, label string) {
	ResourcesToYAMLFile(info, schema.GroupVersionResource{
		Group:    corev1.SchemeGroupVersion.Group,
		Version:  corev1.SchemeGroupVersion.Version,
		Resource: "services",
	}, corev1.NamespaceAll, metav1.ListOptions{LabelSelector: label})
}

func gatherConfigMapLighthouseDNS(info *Info, namespace string) {
	gatherConfigMaps(info, namespace, metav1.ListOptions{LabelSelector: lighthouseComponentsLabel})
}

func isCoreDNSTypeOcp(info *Info) bool {
	pods, err := findPods(info.ClientProducer.ForKubernetes(), ocpCoreDNSPodLabel)
	return err == nil && len(pods.Items) > 0
}
