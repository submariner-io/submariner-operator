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

package network

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner/pkg/cni"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

//nolint:nilnil // Intentional as the purpose is to discover.
func discoverFlannelNetwork(ctx context.Context, client controllerClient.Client) (*ClusterNetwork, error) {
	daemonsets := &appsv1.DaemonSetList{}

	// List all DaemonSets across all namespaces that have the label "k8s-app=flannel"
	// This is used to identify if the cluster is running Flannel as its CNI plugin.
	err := client.List(ctx, daemonsets, controllerClient.MatchingLabels{"k8s-app": "flannel"})
	if err != nil {
		return nil, errors.WithMessage(err, "error listing the Daemonsets for flannel discovery")
	}

	var flannelDaemonSet *appsv1.DaemonSet
	volumes := make([]corev1.Volume, 0)

	// Find the first DaemonSet with "flannel" in its name and store its reference and volumes.
	for i := range daemonsets.Items {
		ds := &daemonsets.Items[i]
		if strings.Contains(ds.Name, "flannel") {
			flannelDaemonSet = ds
			volumes = ds.Spec.Template.Spec.Volumes

			break
		}
	}

	if flannelDaemonSet == nil || len(volumes) < 1 {
		return nil, nil
	}

	// Extract the ConfigMap name from the DaemonSet's volumes and check in the same namespace
	configMapName := findFlannelConfigMapName(volumes)

	clusterNetwork, err := extractCIDRsFromFlannelConfigMap(ctx, client, configMapName, flannelDaemonSet.Namespace)
	if err != nil {
		return nil, err
	}

	if clusterNetwork == nil {
		return nil, errors.New("cluster network is nil")
	}

	clusterNetwork.NetworkPlugin = cni.Flannel

	return clusterNetwork, nil
}

//nolint:nilnil // Intentional as the purpose is to discover.
func extractCIDRsFromFlannelConfigMap(ctx context.Context, client controllerClient.Client, configMapName, namespace string,
) (*ClusterNetwork, error) {
	var podCIDR *string

	if configMapName == "" {
		podIPRange, err := findPodIPRange(ctx, client)
		if err != nil {
			return nil, err
		}

		podCIDR = &podIPRange
	} else {
		// Look for the ConfigMap in the specified namespace
		cm := &corev1.ConfigMap{}

		err := client.Get(ctx, controllerClient.ObjectKey{
			Namespace: namespace,
			Name:      configMapName,
		}, cm)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, nil
			}

			return nil, errors.WithMessagef(err, "error retrieving the flannel ConfigMap %q", configMapName)
		}

		podCIDR = extractPodCIDRFromNetConfigJSON(cm)
		if podCIDR == nil {
			return nil, nil
		}
	}

	clusterNetwork := &ClusterNetwork{
		PodCIDRs: []string{*podCIDR},
	}

	// Try to detect the service CIDRs using the generic functions
	clusterIPRange, err := findClusterIPRange(ctx, client)
	if err != nil {
		return nil, err
	}

	if clusterIPRange != "" {
		clusterNetwork.ServiceCIDRs = []string{clusterIPRange}
	}

	return clusterNetwork, nil
}

func findFlannelConfigMapName(volumes []corev1.Volume) string {
	for i := range volumes {
		if strings.Contains(volumes[i].Name, "flannel") && volumes[i].ConfigMap != nil &&
			volumes[i].ConfigMap.Name != "" {
			return volumes[i].ConfigMap.Name
		}
	}

	return ""
}

func extractPodCIDRFromNetConfigJSON(cm *corev1.ConfigMap) *string {
	netConfJSON := cm.Data["net-conf.json"]
	if netConfJSON == "" {
		return nil
	}
	var netConf struct {
		Network string `json:"Network"`
		// All the other fields are ignored by Unmarshal
	}

	if err := json.Unmarshal([]byte(netConfJSON), &netConf); err == nil {
		return &netConf.Network
	}

	return nil
}
