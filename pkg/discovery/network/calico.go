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

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner/pkg/cni"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// nolint:nilnil // Intentional as the purpose is to discover.
func discoverCalicoNetwork(clientSet kubernetes.Interface) (*ClusterNetwork, error) {
	cmList, err := clientSet.CoreV1().ConfigMaps(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "error listing ConfigMaps")
	}

	findCalicoConfigMap := false

	for i := range cmList.Items {
		if cmList.Items[i].Name == "calico-config" {
			findCalicoConfigMap = true
			break
		}
	}

	if !findCalicoConfigMap {
		return nil, nil
	}

	clusterNetwork, err := discoverNetwork(clientSet)
	if err != nil {
		return nil, err
	}

	if clusterNetwork != nil {
		clusterNetwork.NetworkPlugin = cni.Calico
		return clusterNetwork, nil
	}

	return nil, nil
}
