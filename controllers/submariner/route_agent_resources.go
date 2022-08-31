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
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/helpers"
	"github.com/submariner-io/submariner-operator/pkg/images"
	"github.com/submariner-io/submariner-operator/pkg/names"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

// nolint:wrapcheck // No need to wrap errors here.
func (r *Reconciler) reconcileRouteagentDaemonSet(instance *v1alpha1.Submariner, reqLogger logr.Logger) (*appsv1.DaemonSet,
	error,
) {
	return helpers.ReconcileDaemonSet(instance, newRouteAgentDaemonSet(instance, names.RouteAgentComponent), reqLogger, r.config.ScopedClient,
		r.config.Scheme)
}

func newRouteAgentDaemonSet(cr *v1alpha1.Submariner, name string) *appsv1.DaemonSet {
	labels := map[string]string{
		"app":       name,
		"component": "routeagent",
	}

	maxUnavailable := intstr.FromString("100%")

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"app": name,
			}},
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
					TerminationGracePeriodSeconds: pointer.Int64(1),
					Volumes: []corev1.Volume{
						// We need to share /run/xtables.lock with the host for iptables
						{Name: "host-run-xtables-lock", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
							Path: "/run/xtables.lock",
						}}},
						// We need to share /run/openvswitch/db.sock with the host for OVS
						{Name: "host-run-openvswitch-db-sock", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
							Path: "/run/openvswitch/db.sock",
						}}},
						{Name: "host-sys", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
							Path: "/sys",
						}}},
					},
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           getImagePath(cr, names.RouteAgentImage, names.RouteAgentComponent),
							ImagePullPolicy: images.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.RouteAgentComponent]),
							// FIXME: Should be entrypoint script, find/use correct file for routeagent
							Command: []string{"submariner-route-agent.sh"},
							SecurityContext: &corev1.SecurityContext{
								Capabilities:             &corev1.Capabilities{Add: []corev1.Capability{"ALL"}},
								AllowPrivilegeEscalation: pointer.Bool(true),
								Privileged:               pointer.Bool(true),
								ReadOnlyRootFilesystem:   pointer.Bool(false),
								RunAsNonRoot:             pointer.Bool(false),
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "host-sys", MountPath: "/sys", ReadOnly: true},
								{Name: "host-run-xtables-lock", MountPath: "/run/xtables.lock"},
								{Name: "host-run-openvswitch-db-sock", MountPath: "/run/openvswitch/db.sock"},
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
					ServiceAccountName: names.RouteAgentComponent,
					HostNetwork:        true,
					DNSPolicy:          corev1.DNSClusterFirstWithHostNet,
					// The route agent engine on all nodes, regardless of existing taints
					Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
				},
			},
		},
	}
}
