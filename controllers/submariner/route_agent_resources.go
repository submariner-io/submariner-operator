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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/helpers"
	"github.com/submariner-io/submariner-operator/pkg/names"
)

func (r *SubmarinerReconciler) reconcileRouteagentDaemonSet(instance *v1alpha1.Submariner, reqLogger logr.Logger) (*appsv1.DaemonSet,
	error) {
	return helpers.ReconcileDaemonSet(instance, newRouteAgentDaemonSet(instance), reqLogger, r.client, r.scheme)
}

func newRouteAgentDaemonSet(cr *v1alpha1.Submariner) *appsv1.DaemonSet {
	labels := map[string]string{
		"app":       "submariner-routeagent",
		"component": "routeagent",
	}

	matchLabels := map[string]string{
		"app": "submariner-routeagent",
	}

	allowPrivilegeEscalation := true
	privileged := true
	readOnlyFileSystem := false
	runAsNonRoot := false
	securityContextAllCapAllowEscal := corev1.SecurityContext{
		Capabilities:             &corev1.Capabilities{Add: []corev1.Capability{"ALL"}},
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		Privileged:               &privileged,
		ReadOnlyRootFilesystem:   &readOnlyFileSystem,
		RunAsNonRoot:             &runAsNonRoot,
	}

	terminationGracePeriodSeconds := int64(1)
	maxUnavailable := intstr.FromString("100%")

	routeAgentDaemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      "submariner-routeagent",
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: matchLabels},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &maxUnavailable,
				},
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Volumes: []corev1.Volume{
						{Name: "host-run", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
							Path: "/run",
						}}},
						{Name: "host-sys", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
							Path: "/sys",
						}}},
					},
					Containers: []corev1.Container{
						{
							Name:            "submariner-routeagent",
							Image:           getImagePath(cr, names.RouteAgentImage, names.RouteAgentComponent),
							ImagePullPolicy: helpers.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.RouteAgentComponent]),
							// FIXME: Should be entrypoint script, find/use correct file for routeagent
							Command:         []string{"submariner-route-agent.sh"},
							SecurityContext: &securityContextAllCapAllowEscal,
							VolumeMounts: []corev1.VolumeMount{
								{Name: "host-sys", MountPath: "/sys", ReadOnly: true},
								{Name: "host-run", MountPath: "/run"},
							},
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
					ServiceAccountName: "submariner-routeagent",
					HostNetwork:        true,
					// The route agent engine on all nodes, regardless of existing taints
					Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
				},
			},
		},
	}

	return routeAgentDaemonSet
}
