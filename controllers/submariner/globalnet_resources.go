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
	"github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/helpers"
	"github.com/submariner-io/submariner-operator/controllers/metrics"
	"github.com/submariner-io/submariner-operator/pkg/names"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// nolint:wrapcheck // No need to wrap errors here.
func (r *Reconciler) reconcileGlobalnetDeployment(ctx context.Context, instance *v1alpha1.Submariner,
	reqLogger logr.Logger,
) (*appsv1.Deployment, error) {
	// Moved Globalnet from DaemonSet to Deployment. Make sure the DaemonSet is cleaned up
	r.ensureGlobalnetDaemonSetDeleted(ctx, instance)

	deployment, err := helpers.ReconcileDeployment(instance, newGlobalnetDeployment(instance, names.GlobalnetComponent), reqLogger,
		r.config.Client, r.config.Scheme)
	if err != nil {
		return nil, err
	}

	err = metrics.Setup(instance.Namespace, instance, deployment.GetLabels(), globalnetMetricsServerPort,
		r.config.Client, r.config.RestConfig, r.config.Scheme, reqLogger)

	return deployment, err
}

func newGlobalnetDeployment(cr *v1alpha1.Submariner, name string) *appsv1.Deployment {
	labels := map[string]string{
		"app":       name,
		"component": "globalnet",
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"app": name,
			}},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Replicas: pointer.Int32(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{Name: "host-run-xtables-lock", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
							Path: "/run/xtables.lock",
						}}},
					},
					//nolint:dupl //false positive - lines are similar but not duplicated
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           getImagePath(cr, names.GlobalnetImage, names.GlobalnetImage),
							ImagePullPolicy: helpers.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.GlobalnetImage]),
							SecurityContext: &corev1.SecurityContext{
								Capabilities:             &corev1.Capabilities{Add: []corev1.Capability{"ALL"}},
								AllowPrivilegeEscalation: pointer.Bool(true),
								Privileged:               pointer.Bool(true),
								ReadOnlyRootFilesystem:   pointer.Bool(false),
								RunAsNonRoot:             pointer.Bool(false),
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "host-run-xtables-lock", MountPath: "/run/xtables.lock"},
							},
							Env: []corev1.EnvVar{
								{Name: "SUBMARINER_NAMESPACE", Value: cr.Spec.Namespace},
								{Name: "SUBMARINER_CLUSTERID", Value: cr.Spec.ClusterID},
								{Name: "SUBMARINER_MULTIACTIVEGATEWAYENABLED", Value: strconv.FormatBool(cr.Spec.MultiActiveGatewayEnabled)},
								{Name: "SUBMARINER_EXCLUDENS", Value: "submariner-operator,kube-system,operators,openshift-monitoring,openshift-dns"},
								{Name: "NODE_NAME", ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "spec.nodeName",
									},
								}},
							},
						},
					},
					ServiceAccountName:            names.GlobalnetComponent,
					TerminationGracePeriodSeconds: pointer.Int64(2),
					// The Globalnet Pod must be able to run on any flagged node, regardless of existing taints
					Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
				},
			},
		},
	}
}

// Deprecated: DaemonSet deprecated in favor of a Deployment. Still here for cleanup
// to move from a DaemonSet to a Deployment.
func newGlobalnetDaemonSet(cr *v1alpha1.Submariner, name string) *appsv1.DaemonSet {
	labels := map[string]string{
		"app":       name,
		"component": "globalnet",
	}

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
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{Name: "host-run-xtables-lock", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
							Path: "/run/xtables.lock",
						}}},
					},
					//nolint:dupl //false positive - lines are similar but not duplicated
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           getImagePath(cr, names.GlobalnetImage, names.GlobalnetImage),
							ImagePullPolicy: helpers.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.GlobalnetImage]),
							SecurityContext: &corev1.SecurityContext{
								Capabilities:             &corev1.Capabilities{Add: []corev1.Capability{"ALL"}},
								AllowPrivilegeEscalation: pointer.Bool(true),
								Privileged:               pointer.Bool(true),
								ReadOnlyRootFilesystem:   pointer.Bool(false),
								RunAsNonRoot:             pointer.Bool(false),
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "host-run-xtables-lock", MountPath: "/run/xtables.lock"},
							},
							Env: []corev1.EnvVar{
								{Name: "SUBMARINER_NAMESPACE", Value: cr.Spec.Namespace},
								{Name: "SUBMARINER_CLUSTERID", Value: cr.Spec.ClusterID},
								{Name: "SUBMARINER_MULTIACTIVEGATEWAYENABLED", Value: strconv.FormatBool(cr.Spec.MultiActiveGatewayEnabled)},
								{Name: "SUBMARINER_EXCLUDENS", Value: "submariner-operator,kube-system,operators,openshift-monitoring,openshift-dns"},
								{Name: "NODE_NAME", ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "spec.nodeName",
									},
								}},
							},
						},
					},
					ServiceAccountName:            names.GlobalnetComponent,
					TerminationGracePeriodSeconds: pointer.Int64(2),
					NodeSelector:                  map[string]string{"submariner.io/gateway": "true"},
					HostNetwork:                   true,
					// The Globalnet Pod must be able to run on any flagged node, regardless of existing taints
					Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
				},
			},
		},
	}
}
