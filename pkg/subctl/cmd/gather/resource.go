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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

var fileNameRegexp = regexp.MustCompile(`[<>:"/\|?*]`)

// nolint:gocritic // hugeParam: listOptions - match K8s API.
func ResourcesToYAMLFile(info *Info, ofType schema.GroupVersionResource, namespace string, listOptions metav1.ListOptions) {
	err := func() error {
		list, err := info.ClientProducer.ForDynamic().Resource(ofType).Namespace(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.WithMessagef(err, "error listing %q", ofType.Resource)
		}

		selectorStr := ""
		if listOptions.LabelSelector != "" {
			selectorStr = fmt.Sprintf("by label selector %q ", listOptions.LabelSelector)
		} else if listOptions.FieldSelector != "" {
			selectorStr = fmt.Sprintf("by field selector %q ", listOptions.FieldSelector)
		}

		info.Status.Success("Found %d %s %sin namespace %q", len(list.Items), ofType.Resource,
			selectorStr, namespace)

		for i := range list.Items {
			item := &list.Items[i]

			name := escapeFileName(info.ClusterName+"_"+ofType.Resource+"_"+item.GetNamespace()+"_"+item.GetName()) + ".yaml"
			path := filepath.Join(info.DirName, name)

			file, err := os.Create(path)
			if err != nil {
				return errors.WithMessagef(err, "error opening file %s", path)
			}

			defer file.Close()

			data, err := yaml.Marshal(item)
			if err != nil {
				return errors.WithMessage(err, "error marshaling to YAML")
			}

			scrubbedData := scrubSensitiveData(info, string(data))

			_, err = file.Write([]byte(scrubbedData))
			if err != nil {
				return errors.WithMessagef(err, "error writing to file %s", path)
			}

			info.Summary.Resources = append(info.Summary.Resources, ResourceInfo{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
				Type:      ofType.Resource,
				FileName:  name,
			})
		}

		return nil
	}()
	if err != nil {
		info.Status.Failure("Failed to gather %s: %s", ofType.Resource, err)
	}
}

// nolint:gocritic // hugeParam: listOptions - match K8s API.
func gatherDaemonSet(info *Info, namespace string, listOptions metav1.ListOptions) {
	ResourcesToYAMLFile(info, schema.GroupVersionResource{
		Group:    appsv1.SchemeGroupVersion.Group,
		Version:  appsv1.SchemeGroupVersion.Version,
		Resource: "daemonsets",
	}, namespace, listOptions)
}

// nolint:gocritic // hugeParam: listOptions - match K8s API.
func gatherDeployment(info *Info, namespace string, listOptions metav1.ListOptions) {
	ResourcesToYAMLFile(info, schema.GroupVersionResource{
		Group:    appsv1.SchemeGroupVersion.Group,
		Version:  appsv1.SchemeGroupVersion.Version,
		Resource: "deployments",
	}, namespace, listOptions)
}

// nolint:gocritic // hugeParam: listOptions - match K8s API.
func gatherConfigMaps(info *Info, namespace string, listOptions metav1.ListOptions) {
	ResourcesToYAMLFile(info, schema.GroupVersionResource{
		Group:    corev1.SchemeGroupVersion.Group,
		Version:  corev1.SchemeGroupVersion.Version,
		Resource: "configmaps",
	}, namespace, listOptions)
}

func scrubSensitiveData(info *Info, dataString string) string {
	if info.IncludeSensitiveData {
		return dataString
	}

	if info.Submariner != nil {
		dataString = replaceIfNotEmpty(dataString, info.Submariner.Spec.BrokerK8sApiServer, "##redacted-api-server##")
		dataString = replaceIfNotEmpty(dataString, info.Submariner.Spec.BrokerK8sApiServerToken, "##redacted-token##")
		dataString = replaceIfNotEmpty(dataString, info.Submariner.Spec.BrokerK8sCA, "##redacted-ca##")
		dataString = replaceIfNotEmpty(dataString, info.Submariner.Spec.CeIPSecPSK, "##redacted-ipsec-psk##")
	} else if info.ServiceDiscovery != nil {
		dataString = replaceIfNotEmpty(dataString, info.ServiceDiscovery.Spec.BrokerK8sApiServer, "##redacted-api-server##")
		dataString = replaceIfNotEmpty(dataString, info.ServiceDiscovery.Spec.BrokerK8sApiServerToken, "##redacted-token##")
		dataString = replaceIfNotEmpty(dataString, info.ServiceDiscovery.Spec.BrokerK8sCA, "##redacted-ca##")
	}

	return dataString
}

func escapeFileName(s string) string {
	return fileNameRegexp.ReplaceAllString(s, "_")
}

func replaceIfNotEmpty(s, old, replacement string) string {
	if old == "" {
		return s
	}

	return strings.ReplaceAll(s, old, replacement)
}
