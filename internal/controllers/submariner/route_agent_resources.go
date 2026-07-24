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
	"maps"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/submariner-io/admiral/pkg/names"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/controllers/apply"
	"github.com/submariner-io/submariner-operator/pkg/ciliumcm"
	"github.com/submariner-io/submariner-operator/pkg/httpproxy"
	"github.com/submariner-io/submariner-operator/pkg/images"
	opnames "github.com/submariner-io/submariner-operator/pkg/names"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
	healthCheckInterval := uint(1)
	healthCheckMaxPacketLossCount := uint(5)

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
					TerminationGracePeriodSeconds: new(int64(1)),
					Volumes: []corev1.Volume{
						// Share /run/xtables.lock with the host for iptables
						{Name: "host-run-xtables-lock", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
							Path: "/run/xtables.lock",
						}}},
						// Share /run/openvswitch/db.sock and /run/openvswitch/ovnnb_db.sock with the host for OVS/OVN
						{Name: "host-run-openvswitch", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
							Path: "/run/openvswitch", Type: new(corev1.HostPathDirectoryOrCreate),
						}}},
						// Share /sys with the host for OVS/OVN
						{Name: "host-sys", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
							Path: "/sys",
						}}},
						// Share /run/ovn-ic with the host for OVN (this is a transitional path used by OpenShift for upgrades)
						{Name: "host-run-ovn-ic", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
							Path: "/run/ovn-ic", Type: new(corev1.HostPathDirectoryOrCreate),
						}}},
					},
					// The route agent needs to wait for the node to be ready before starting,
					// to avoid racing with the CNI for socket setup; this init container takes care of that
					InitContainers: []corev1.Container{
						{
							Name:            name + "-init",
							Image:           getImagePath(cr, opnames.RouteAgentImage, names.RouteAgentComponent),
							ImagePullPolicy: images.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.RouteAgentComponent]),
							Command:         []string{"await-node-ready.sh"},
							Env: httpproxy.AddEnvVars([]corev1.EnvVar{
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
								AllowPrivilegeEscalation: new(true),
								Privileged:               new(true),
								ReadOnlyRootFilesystem:   new(false),
								RunAsNonRoot:             new(false),
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
								{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.name",
									},
								}},
								{Name: "SUBMARINER_HEALTHCHECKENABLED", Value: strconv.FormatBool(healthCheckEnabled)},
								{Name: "SUBMARINER_HEALTHCHECKINTERVAL", Value: strconv.FormatUint(uint64(healthCheckInterval), 10)},
								{Name: "SUBMARINER_HEALTHCHECKMAXPACKETLOSSCOUNT", Value: strconv.FormatUint(uint64(healthCheckMaxPacketLossCount), 10)},
								{Name: "SUBMARINER_INTRAROUTINGDISABLED", Value: strconv.FormatBool(cr.Spec.DisableIntraClusterConnectivity)},
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

	addCiliumCMPublisherToRouteAgent(ds, cr)

	// Apply node selector from spec if configured.
	if len(cr.Spec.NodeSelector) > 0 {
		if ds.Spec.Template.Spec.NodeSelector == nil {
			ds.Spec.Template.Spec.NodeSelector = make(map[string]string)
		}

		maps.Copy(ds.Spec.Template.Spec.NodeSelector, cr.Spec.NodeSelector)
	}

	// When intra-cluster connectivity is disabled, also restrict to gateway nodes.
	if cr.Spec.DisableIntraClusterConnectivity {
		if ds.Spec.Template.Spec.NodeSelector == nil {
			ds.Spec.Template.Spec.NodeSelector = map[string]string{}
		}

		ds.Spec.Template.Spec.NodeSelector["submariner.io/gateway"] = "true"
	}

	return ds
}

func addCiliumCMPublisherToRouteAgent(ds *appsv1.DaemonSet, cr *v1alpha1.Submariner) {
	if cr.Status.NetworkPlugin != "cilium" {
		return
	}

	ds.Spec.Template.Spec.Volumes = append(ds.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: ciliumcm.VolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: ciliumcm.TLSSecretName,
				// Project only what the publisher needs: never mount ca.key
				// (CA signing key) or client certs into every node.
				Items: []corev1.KeyToPath{
					{Key: ciliumcm.TLSCertKey, Path: ciliumcm.TLSCertKey},
					{Key: ciliumcm.TLSKeyKey, Path: ciliumcm.TLSKeyKey},
					{Key: ciliumcm.CACertKey, Path: ciliumcm.CACertKey},
				},
			},
		},
	})

	if len(ds.Spec.Template.Spec.Containers) == 0 {
		return
	}

	container := &ds.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      ciliumcm.VolumeName,
		MountPath: ciliumcm.MountPath,
		ReadOnly:  true,
	})

	container.Env = append(container.Env,
		corev1.EnvVar{Name: "SUBMARINER_CILIUM_CM_CERT_FILE", Value: ciliumcm.MountPath + "/" + ciliumcm.TLSCertKey},
		corev1.EnvVar{Name: "SUBMARINER_CILIUM_CM_KEY_FILE", Value: ciliumcm.MountPath + "/" + ciliumcm.TLSKeyKey},
		corev1.EnvVar{Name: "SUBMARINER_CILIUM_CM_CA_FILE", Value: ciliumcm.MountPath + "/" + ciliumcm.CACertKey},
		corev1.EnvVar{Name: "SUBMARINER_CILIUM_CM_LISTEN_URL", Value: ciliumcm.DefaultListenURL},
		corev1.EnvVar{Name: "SUBMARINER_CILIUM_CM_PEER_URL", Value: ciliumcm.DefaultPeerURL},
		corev1.EnvVar{Name: "SUBMARINER_CILIUM_CM_REMOTE_NAME", Value: ciliumcm.DefaultRemoteName},
		corev1.EnvVar{Name: "SUBMARINER_CILIUM_CM_CLUSTER_ID", Value: ciliumcm.DefaultClusterID},
	)
}
