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
	"context"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/submariner-io/admiral/pkg/names"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/apply"
	"github.com/submariner-io/submariner-operator/pkg/httpproxy"
	"github.com/submariner-io/submariner-operator/pkg/images"
	opnames "github.com/submariner-io/submariner-operator/pkg/names"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

//nolint:wrapcheck // No need to wrap errors here.
func (r *Reconciler) reconcileRouteagentDaemonSet(ctx context.Context, instance *v1alpha1.Submariner,
	reqLogger logr.Logger,
) (*appsv1.DaemonSet, error) {
	return apply.DaemonSet(ctx, instance, newRouteAgentDaemonSet(instance, names.RouteAgentComponent),
		reqLogger, r.config.ScopedClient, r.config.Scheme)
}

func newRouteAgentDaemonSet(cr *v1alpha1.Submariner, name string) *appsv1.DaemonSet {
	// Default healthCheck Values
	healthCheckEnabled := true
	// The values are in seconds
	healthCheckInterval := uint64(1)
	healthCheckMaxPacketLossCount := uint64(5)

	if cr.Spec.ConnectionHealthCheck != nil {
		healthCheckEnabled = cr.Spec.ConnectionHealthCheck.Enabled
		healthCheckInterval = cr.Spec.ConnectionHealthCheck.IntervalSeconds
		healthCheckMaxPacketLossCount = cr.Spec.ConnectionHealthCheck.MaxPacketLossCount
	}

	labels := map[string]string{
		"app":       name,
		"component": "routeagent",
	}
	maxUnavailable := intstr.FromString("100%")

	ds := &appsv1.DaemonSet{
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
					TerminationGracePeriodSeconds: ptr.To(int64(1)),
					Volumes: []corev1.Volume{
						// Share /run/xtables.lock with the host for iptables
						{Name: "host-run-xtables-lock", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
							Path: "/run/xtables.lock",
						}}},
						// Share /run/openvswitch/db.sock and /run/openvswitch/ovnnb_db.sock with the host for OVS/OVN
						{Name: "host-run-openvswitch", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
							Path: "/run/openvswitch", Type: ptr.To(corev1.HostPathDirectoryOrCreate),
						}}},
						// Share /sys with the host for OVS/OVN
						{Name: "host-sys", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
							Path: "/sys",
						}}},
						// Share /run/ovn-ic with the host for OVN (this is a transitional path used by OpenShift for upgrades)
						{Name: "host-run-ovn-ic", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
							Path: "/run/ovn-ic", Type: ptr.To(corev1.HostPathDirectoryOrCreate),
						}}},
					},
					// The route agent needs to wait for the node to be ready before starting,
					// to avoid racing with the CNI for socket setup; this init container takes care of that
					InitContainers: []corev1.Container{
						{
							Name:            name + "-init",
							Image:           getImagePath(cr, opnames.RouteAgentImage, names.RouteAgentComponent),
							ImagePullPolicy: images.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.RouteAgentComponent]),
							Env: httpproxy.AddEnvVars([]corev1.EnvVar{
								{Name: "SUBMARINER_WAITFORNODE", Value: "true"},
								{Name: "NODE_NAME", ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "spec.nodeName",
									},
								}},
							}),
						},
					},
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           getImagePath(cr, opnames.RouteAgentImage, names.RouteAgentComponent),
							ImagePullPolicy: images.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.RouteAgentComponent]),
							SecurityContext: &corev1.SecurityContext{
								Capabilities:             &corev1.Capabilities{Add: []corev1.Capability{"ALL"}},
								AllowPrivilegeEscalation: ptr.To(true),
								Privileged:               ptr.To(true),
								ReadOnlyRootFilesystem:   ptr.To(false),
								RunAsNonRoot:             ptr.To(false),
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "host-sys", MountPath: "/sys", ReadOnly: true},
								{Name: "host-run-xtables-lock", MountPath: "/run/xtables.lock"},
								{Name: "host-run-openvswitch", MountPath: "/run/openvswitch"},
								{Name: "host-run-ovn-ic", MountPath: "/run/ovn-ic"},
							},
							Env: httpproxy.AddEnvVars([]corev1.EnvVar{
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
								{Name: "SUBMARINER_HEALTHCHECKENABLED", Value: strconv.FormatBool(healthCheckEnabled)},
								{Name: "SUBMARINER_HEALTHCHECKINTERVAL", Value: strconv.FormatUint(healthCheckInterval, 10)},
								{Name: "SUBMARINER_HEALTHCHECKMAXPACKETLOSSCOUNT", Value: strconv.FormatUint(healthCheckMaxPacketLossCount, 10)},
							}),
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

	return ds
}
