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
)

//nolint:wrapcheck // No need to wrap errors here.
func (r *Reconciler) reconcileMetricsProxyDaemonSet(ctx context.Context, instance *v1alpha1.Submariner, reqLogger logr.Logger,
) (*appsv1.DaemonSet, error) {
	return apply.DaemonSet(ctx, instance, newMetricsProxyDaemonSet(instance), reqLogger,
		r.config.ScopedClient, r.config.Scheme)
}

func newMetricsProxyDaemonSet(cr *v1alpha1.Submariner) *appsv1.DaemonSet {
	labels := map[string]string{
		"app":       names.MetricsProxyComponent,
		"component": "metrics",
	}

	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      names.MetricsProxyComponent,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"app": names.MetricsProxyComponent,
			}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						*metricProxyContainer(cr, "gateway-metrics-proxy", strconv.Itoa(gatewayMetricsServicePort),
							gatewayMetricsServerPort),
					},
					NodeSelector: map[string]string{"submariner.io/gateway": "true"},
					// The MetricsProxy Pod must be able to run on any flagged node, regardless of existing taints
					Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
				},
			},
		},
	}

	if cr.Spec.GlobalCIDR != "" {
		daemonSet.Spec.Template.Spec.Containers = append(daemonSet.Spec.Template.Spec.Containers,
			*metricProxyContainer(cr, "globalnet-metrics-proxy", strconv.Itoa(globalnetMetricsServicePort), globalnetMetricsServerPort))
	}

	return daemonSet
}

func metricProxyContainer(cr *v1alpha1.Submariner, name, hostPort, podPort string) *corev1.Container {
	return &corev1.Container{
		Name:            name,
		Image:           getImagePath(cr, opnames.MetricsProxyImage, names.MetricsProxyComponent),
		ImagePullPolicy: images.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.MetricsProxyComponent]),
		Env: httpproxy.AddEnvVars([]corev1.EnvVar{
			{Name: "NODE_IP", ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.hostIP",
				},
			}},
		}),
		Command: []string{"/app/metricsproxy"},
		Args:    []string{hostPort, "$(NODE_IP)", podPort},
	}
}
