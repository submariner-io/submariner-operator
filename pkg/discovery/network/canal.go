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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

//nolint:nilnil // Intentional as the purpose is to discover.
func discoverCanalFlannelNetwork(ctx context.Context, client controllerClient.Client) (*ClusterNetwork, error) {
	daemonsets := &appsv1.DaemonSetList{}

	err := client.List(ctx, daemonsets, controllerClient.InNamespace(metav1.NamespaceSystem),
		controllerClient.MatchingLabelsSelector{Selector: labels.Set{"k8s-app": "canal"}.AsSelector()})
	if err != nil {
		return nil, errors.WithMessage(err, "error listing Daemonsets for canal discovery")
	}

	if len(daemonsets.Items) == 0 {
		return nil, nil
	}

	// Pass the namespace to the function call
	clusterNetwork, err := extractCIDRsFromFlannelConfigMap(ctx, client, findFlannelConfigMapName(
		daemonsets.Items[0].Spec.Template.Spec.Volumes), metav1.NamespaceSystem)
	if err != nil || clusterNetwork == nil {
		return nil, err
	}

	clusterNetwork.NetworkPlugin = cni.CanalFlannel

	return clusterNetwork, nil
}
