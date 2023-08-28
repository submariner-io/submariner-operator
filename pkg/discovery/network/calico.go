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
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

//nolint:nilnil // Intentional as the purpose is to discover.
func discoverCalicoNetwork(ctx context.Context, client controllerClient.Client) (*ClusterNetwork, error) {
	found, err := calicoConfigMapExists(ctx, client)
	if err != nil {
		return nil, err
	}

	if !found {
		found, err = calicoDaemonSetExists(ctx, client)
		if err != nil {
			return nil, err
		}
	}

	if !found {
		return nil, nil
	}

	clusterNetwork, err := discoverNetwork(ctx, client)
	if err != nil {
		return nil, err
	}

	if clusterNetwork != nil {
		clusterNetwork.NetworkPlugin = cni.Calico
		return clusterNetwork, nil
	}

	return nil, nil
}

func calicoConfigMapExists(ctx context.Context, client controllerClient.Client) (bool, error) {
	cmList := &corev1.ConfigMapList{}

	err := client.List(ctx, cmList, controllerClient.InNamespace(metav1.NamespaceAll))
	if err != nil {
		return false, errors.Wrapf(err, "error listing ConfigMaps")
	}

	for i := range cmList.Items {
		if cmList.Items[i].Name == "calico-config" {
			return true, nil
		}
	}

	return false, nil
}

func calicoDaemonSetExists(ctx context.Context, client controllerClient.Client) (bool, error) {
	dsList := &v1.DaemonSetList{}

	err := client.List(ctx, dsList, controllerClient.InNamespace(metav1.NamespaceAll))
	if err != nil {
		return false, errors.Wrapf(err, "error listing DaemonSets")
	}

	for i := range dsList.Items {
		if dsList.Items[i].Name == "calico-node" {
			return true, nil
		}
	}

	return false, nil
}
