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

	"github.com/submariner-io/submariner/pkg/cni"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	ovnKubeService     = "ovnkube-db"
	OvnNBDB            = "OVN_NBDB"
	OvnSBDB            = "OVN_SBDB"
	OvnNBDBDefaultPort = 6641
	OvnSBDBDefaultPort = 6642
)

func discoverOvnKubernetesNetwork(clientSet kubernetes.Interface) (*ClusterNetwork, error) {
	ovnDBPod, err := FindPod(clientSet, "name=ovnkube-db")

	if err != nil || ovnDBPod == nil {
		return nil, err
	}

	if _, err := clientSet.CoreV1().Services(ovnDBPod.Namespace).Get(context.TODO(), ovnKubeService, v1.GetOptions{}); err != nil {
		return nil, fmt.Errorf("error finding %q service in %q namespace", ovnKubeService, ovnDBPod.Namespace)
	}

	dbConnectionProtocol := "tcp"

	for i := range ovnDBPod.Spec.Containers {
		for _, envVar := range ovnDBPod.Spec.Containers[i].Env {
			if envVar.Name == "OVN_SSL_ENABLE" {
				if !strings.EqualFold(envVar.Value, "NO") {
					dbConnectionProtocol = "ssl"
				}
			}
		}
	}

	clusterNetwork := &ClusterNetwork{
		NetworkPlugin: cni.OVNKubernetes,
		PluginSettings: map[string]string{
			OvnNBDB: fmt.Sprintf("%s:%s.%s:%d", dbConnectionProtocol, ovnKubeService, ovnDBPod.Namespace, OvnNBDBDefaultPort),
			OvnSBDB: fmt.Sprintf("%s:%s.%s:%d", dbConnectionProtocol, ovnKubeService, ovnDBPod.Namespace, OvnSBDBDefaultPort),
		},
	}

	// If the cluster/service CIDRs weren't found we leave it to the generic functions to figure out later
	if ovnConfig, err := clientSet.CoreV1().ConfigMaps(ovnDBPod.Namespace).Get(context.TODO(), "ovn-config", v1.GetOptions{}); err == nil {
		if netCidr, ok := ovnConfig.Data["net_cidr"]; ok {
			clusterNetwork.PodCIDRs = []string{netCidr}
		}

		if svcCidr, ok := ovnConfig.Data["svc_cidr"]; ok {
			clusterNetwork.ServiceCIDRs = []string{svcCidr}
		}
	}

	return clusterNetwork, nil
}
