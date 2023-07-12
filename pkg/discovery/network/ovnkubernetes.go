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
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner/pkg/cni"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ovnKubeService     = "ovnkube-db"
	OvnNBDB            = "OVN_NBDB"
	OvnSBDB            = "OVN_SBDB"
	OvnNBDBDefaultPort = 6641
	OvnSBDBDefaultPort = 6642
)

func discoverOvnKubernetesNetwork(ctx context.Context, client controllerClient.Client) (*ClusterNetwork, error) {
	ovnDBPod, err := FindPod(ctx, client, "name=ovnkube-db")
	if err != nil {
		return nil, err
	}

	var clusterNetwork *ClusterNetwork

	if ovnDBPod != nil {
		clusterNetwork, err = discoverOvnDBClusterNetwork(ctx, client, ovnDBPod)
	} else {
		clusterNetwork, err = discoverOvnNodeClusterNetwork(ctx, client)
	}

	if err != nil || clusterNetwork == nil {
		return nil, err
	}

	clusterNetwork.NetworkPlugin = cni.OVNKubernetes

	return clusterNetwork, nil
}

func discoverOvnDBClusterNetwork(ctx context.Context, client controllerClient.Client, ovnDBPod *corev1.Pod) (*ClusterNetwork, error) {
	err := client.Get(ctx, types.NamespacedName{Namespace: ovnDBPod.Namespace, Name: ovnKubeService}, &corev1.Service{})
	if err != nil {
		return nil, fmt.Errorf("error finding %q service in %q namespace", ovnKubeService, ovnDBPod.Namespace)
	}

	dbConnectionProtocol := findProtocol(ovnDBPod)

	clusterNetwork := &ClusterNetwork{
		PluginSettings: map[string]string{
			OvnNBDB: fmt.Sprintf("%s:%s.%s:%d", dbConnectionProtocol, ovnKubeService, ovnDBPod.Namespace, OvnNBDBDefaultPort),
			OvnSBDB: fmt.Sprintf("%s:%s.%s:%d", dbConnectionProtocol, ovnKubeService, ovnDBPod.Namespace, OvnSBDBDefaultPort),
		},
	}

	updateClusterNetworkFromConfigMap(ctx, client, ovnDBPod.Namespace, clusterNetwork)

	return clusterNetwork, nil
}

func discoverOvnNodeClusterNetwork(ctx context.Context, client controllerClient.Client) (*ClusterNetwork, error) {
	// In OVN IC deployments, the ovn DB will be a part of ovnkube-node
	ovnPod, err := FindPod(ctx, client, "name=ovnkube-node")
	if err != nil || ovnPod == nil {
		return nil, err
	}

	endpointList, err := FindEndpoint(ctx, client, ovnPod.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "Error retrieving the endpoints from namespace %q", ovnPod.Namespace)
	}

	var clusterNetwork *ClusterNetwork

	if endpointList == nil || len(endpointList.Items) == 0 {
		clusterNetwork, err = createLocalClusterNetwork(), nil
	} else {
		clusterNetwork, err = createClusterNetworkWithEndpoints(endpointList.Items), nil
	}

	if err != nil {
		return nil, err
	}

	updateClusterNetworkFromConfigMap(ctx, client, ovnPod.Namespace, clusterNetwork)

	return clusterNetwork, nil
}

func createLocalClusterNetwork() *ClusterNetwork {
	return &ClusterNetwork{
		PluginSettings: map[string]string{
			OvnNBDB: "local",
			OvnSBDB: "local",
		},
	}
}

func createClusterNetworkWithEndpoints(endPoints []corev1.Endpoints) *ClusterNetwork {
	pluginSettings := map[string]string{}
	var OvnNBDBIPs, OVNSBDBIps string

	for index := 0; index < len(endPoints); index++ {
		for _, subset := range endPoints[index].Subsets {
			for _, port := range subset.Ports {
				if strings.Contains(port.Name, "north") {
					OvnNBDBIPs += fmt.Sprintf("%s:%s:%s:%s:%d,",
						"IC:", endPoints[index].Name, port.Protocol, subset.Addresses[0].IP, OvnNBDBDefaultPort)
				} else if strings.Contains(port.Name, "south") {
					OVNSBDBIps += fmt.Sprintf("%s:%s:%s:%s:%d,",
						"IC:", endPoints[index].Name, port.Protocol, subset.Addresses[0].IP, OvnSBDBDefaultPort)
				}
			}
		}
	}

	pluginSettings[OvnNBDB] = OvnNBDBIPs
	pluginSettings[OvnSBDB] = OVNSBDBIps

	return &ClusterNetwork{
		PluginSettings: pluginSettings,
	}
}

func updateClusterNetworkFromConfigMap(ctx context.Context, client controllerClient.Client, ovnPodNamespace string,
	clusterNetwork *ClusterNetwork,
) {
	ovnConfig := &corev1.ConfigMap{}

	err := client.Get(ctx, types.NamespacedName{Namespace: ovnPodNamespace, Name: "ovn-config"}, ovnConfig)
	if err == nil {
		if netCidr, ok := ovnConfig.Data["net_cidr"]; ok {
			clusterNetwork.PodCIDRs = []string{netCidr}
		}

		if svcCidr, ok := ovnConfig.Data["svc_cidr"]; ok {
			clusterNetwork.ServiceCIDRs = []string{svcCidr}
		}
	}
}

func FindEndpoint(ctx context.Context, client controllerClient.Client, endpointNameSpace string) (*corev1.EndpointsList, error) {
	endpointsList := &corev1.EndpointsList{}
	listOptions := &controllerClient.ListOptions{
		Namespace: endpointNameSpace,
	}

	err := client.List(ctx, endpointsList, listOptions)

	return endpointsList, errors.WithMessagef(err, "error listing endpoints in namespace %q", endpointNameSpace)
}

func findProtocol(pod *corev1.Pod) string {
	dbConnectionProtocol := "tcp"

	for i := range pod.Spec.Containers {
		for _, envVar := range pod.Spec.Containers[i].Env {
			if envVar.Name == "OVN_SSL_ENABLE" {
				if !strings.EqualFold(envVar.Value, "NO") {
					dbConnectionProtocol = "ssl"
				}
			}
		}
	}

	return dbConnectionProtocol
}
