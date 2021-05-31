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

package submariner

import (
	"strconv"

	"github.com/go-logr/logr"
	"github.com/submariner-io/submariner/pkg/routeagent_driver/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/helpers"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/names"
)

func (r *SubmarinerReconciler) reconcileNetworkPluginSyncerDeployment(instance *v1alpha1.Submariner,
	clusterNetwork *network.ClusterNetwork, reqLogger logr.Logger) (*appsv1.Deployment, error) {
	// Only OVNKubernetes needs networkplugin-syncer so far
	if instance.Status.NetworkPlugin == constants.NetworkPluginOVNKubernetes {
		return helpers.ReconcileDeployment(instance, newNetworkPluginSyncerDeployment(instance,
			clusterNetwork), reqLogger, r.client, r.scheme)
	}
	return nil, nil
}

func newNetworkPluginSyncerDeployment(cr *v1alpha1.Submariner, clusterNetwork *network.ClusterNetwork) *appsv1.Deployment {
	labels := map[string]string{
		"app":       "submariner-networkplugin-syncer",
		"component": "networkplugin-syncer",
	}

	matchLabels := map[string]string{
		"app": "submariner-networkplugin-syncer",
	}

	nReplicas := int32(1)
	terminationGracePeriodSeconds := int64(1)

	networkPluginSyncerDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      "submariner-networkplugin-syncer",
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: matchLabels},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Replicas: &nReplicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Containers: []corev1.Container{
						{
							Name:            "submariner-routeagent",
							Image:           getImagePath(cr, names.NetworkPluginSyncerImage, names.NetworkPluginSyncerComponent),
							ImagePullPolicy: helpers.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.NetworkPluginSyncerComponent]),
							Command:         []string{"submariner-networkplugin-syncer.sh"},
							Env: []corev1.EnvVar{
								{Name: "SUBMARINER_NAMESPACE", Value: cr.Spec.Namespace},
								{Name: "SUBMARINER_CLUSTERID", Value: cr.Spec.ClusterID},
								{Name: "SUBMARINER_DEBUG", Value: strconv.FormatBool(cr.Spec.Debug)},
								{Name: "SUBMARINER_CLUSTERCIDR", Value: cr.Status.ClusterCIDR},
								{Name: "SUBMARINER_SERVICECIDR", Value: cr.Status.ServiceCIDR},
								{Name: "SUBMARINER_GLOBALCIDR", Value: cr.Spec.GlobalCIDR},
								{Name: "SUBMARINER_NETWORKPLUGIN", Value: cr.Status.NetworkPlugin},
								{Name: "NODE_NAME", ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "spec.nodeName",
									},
								}},
							},
						},
					},
					ServiceAccountName: "submariner-networkplugin-syncer",
					Tolerations:        []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
				},
			},
		},
	}

	if clusterNetwork.PluginSettings != nil {
		if ovndb, ok := clusterNetwork.PluginSettings[network.OvnNBDB]; ok {
			networkPluginSyncerDeployment.Spec.Template.Spec.Containers[0].Env =
				append(networkPluginSyncerDeployment.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
					Name: network.OvnNBDB, Value: ovndb})
		}
		if ovnsb, ok := clusterNetwork.PluginSettings[network.OvnSBDB]; ok {
			networkPluginSyncerDeployment.Spec.Template.Spec.Containers[0].Env =
				append(networkPluginSyncerDeployment.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
					Name: network.OvnSBDB, Value: ovnsb})
		}
	}

	return networkPluginSyncerDeployment
}
