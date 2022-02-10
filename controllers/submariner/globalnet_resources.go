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
	"github.com/go-logr/logr"
	"github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/helpers"
	"github.com/submariner-io/submariner-operator/controllers/metrics"
	"github.com/submariner-io/submariner-operator/pkg/names"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// nolint:wrapcheck // No need to wrap errors here.
func (r *Reconciler) reconcileGlobalnetDaemonSet(instance *v1alpha1.Submariner, reqLogger logr.Logger) (*appsv1.DaemonSet,
	error) {
	daemonSet, err := helpers.ReconcileDaemonSet(instance, newGlobalnetDaemonSet(instance, names.GlobalnetComponent), reqLogger,
		r.config.Client, r.config.Scheme)
	if err != nil {
		return nil, err
	}

	err = metrics.Setup(instance.Namespace, instance, daemonSet.GetLabels(), globalnetMetricsServerPort,
		r.config.Client, r.config.RestConfig, r.config.Scheme, reqLogger)

	return daemonSet, err
}

func newGlobalnetDaemonSet(cr *v1alpha1.Submariner, name string) *appsv1.DaemonSet {
	labels := map[string]string{
		"app":       name,
		"component": "globalnet",
	}

	matchLabels := map[string]string{
		"app": name,
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

	terminationGracePeriodSeconds := int64(2)

	globalnetDaemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: matchLabels},
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
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           getImagePath(cr, names.GlobalnetImage, names.GlobalnetImage),
							ImagePullPolicy: helpers.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.GlobalnetImage]),
							SecurityContext: &securityContextAllCapAllowEscal,
							VolumeMounts: []corev1.VolumeMount{
								{Name: "host-run-xtables-lock", MountPath: "/run/xtables.lock"},
							},
							Env: []corev1.EnvVar{
								{Name: "SUBMARINER_NAMESPACE", Value: cr.Spec.Namespace},
								{Name: "SUBMARINER_CLUSTERID", Value: cr.Spec.ClusterID},
								{Name: "SUBMARINER_EXCLUDENS", Value: "submariner-operator,kube-system,operators,openshift-monitoring,openshift-dns"},
								{Name: "NODE_NAME", ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "spec.nodeName",
									},
								}},
							},
						},
					},
					ServiceAccountName:            name,
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					NodeSelector:                  map[string]string{"submariner.io/gateway": "true"},
					HostNetwork:                   true,
					// The Globalnet Pod must be able to run on any flagged node, regardless of existing taints
					Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
				},
			},
		},
	}

	return globalnetDaemonSet
}
