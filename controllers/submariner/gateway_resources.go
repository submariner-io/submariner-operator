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
	submarinerv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/helpers"
	"github.com/submariner-io/submariner-operator/controllers/metrics"
	"github.com/submariner-io/submariner-operator/pkg/names"
)

func newGatewayDaemonSet(cr *v1alpha1.Submariner) *appsv1.DaemonSet {
	labels := map[string]string{
		"app":       "submariner-gateway",
		"component": "gateway",
	}

	revisionHistoryLimit := int32(5)

	maxUnavailable := intstr.FromInt(1)

	deployment := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Namespace: cr.Namespace,
			Name:      "submariner-gateway",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "submariner-gateway"}},
			Template: newGatewayPodTemplate(cr),
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &maxUnavailable,
				},
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
			},
			RevisionHistoryLimit: &revisionHistoryLimit,
		},
	}

	return deployment
}

// newGatewayPodTemplate returns a submariner pod with the same fields as the cr
func newGatewayPodTemplate(cr *v1alpha1.Submariner) corev1.PodTemplateSpec {
	labels := map[string]string{
		"app": "submariner-gateway",
	}

	// Create privileged security context for Gateway pod
	// FIXME: Seems like these have to be a var, so can pass pointer to bool var to SecurityContext. Cleaner option?
	allowPrivilegeEscalation := true
	privileged := true
	runAsNonRoot := false
	readOnlyRootFilesystem := false

	securityContextAllCapsPrivileged := corev1.SecurityContext{
		Capabilities:             &corev1.Capabilities{Add: []corev1.Capability{"ALL"}},
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		Privileged:               &privileged,
		ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
		RunAsNonRoot:             &runAsNonRoot}

	// Create Pod
	terminationGracePeriodSeconds := int64(1)

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

	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
		Spec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: labels,
						},
						TopologyKey: "kubernetes.io/hostname",
					}},
				},
			},
			NodeSelector: map[string]string{"submariner.io/gateway": "true"},
			Containers: []corev1.Container{
				{
					Name:            "submariner-gateway",
					Image:           getImagePath(cr, names.GatewayImage, names.GatewayComponent),
					ImagePullPolicy: helpers.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.GatewayComponent]),
					Command:         []string{"submariner.sh"},
					SecurityContext: &securityContextAllCapsPrivileged,
					Env: []corev1.EnvVar{
						{Name: "SUBMARINER_NAMESPACE", Value: cr.Spec.Namespace},
						{Name: "SUBMARINER_CLUSTERCIDR", Value: cr.Status.ClusterCIDR},
						{Name: "SUBMARINER_SERVICECIDR", Value: cr.Status.ServiceCIDR},
						{Name: "SUBMARINER_GLOBALCIDR", Value: cr.Spec.GlobalCIDR},
						{Name: "SUBMARINER_CLUSTERID", Value: cr.Spec.ClusterID},
						{Name: "SUBMARINER_COLORCODES", Value: cr.Spec.ColorCodes},
						{Name: "SUBMARINER_DEBUG", Value: strconv.FormatBool(cr.Spec.Debug)},
						{Name: "SUBMARINER_NATENABLED", Value: strconv.FormatBool(cr.Spec.NatEnabled)},
						{Name: "SUBMARINER_BROKER", Value: cr.Spec.Broker},
						{Name: "SUBMARINER_CABLEDRIVER", Value: cr.Spec.CableDriver},
						{Name: "BROKER_K8S_APISERVER", Value: cr.Spec.BrokerK8sApiServer},
						{Name: "BROKER_K8S_APISERVERTOKEN", Value: cr.Spec.BrokerK8sApiServerToken},
						{Name: "BROKER_K8S_REMOTENAMESPACE", Value: cr.Spec.BrokerK8sRemoteNamespace},
						{Name: "BROKER_K8S_CA", Value: cr.Spec.BrokerK8sCA},
						{Name: "CE_IPSEC_PSK", Value: cr.Spec.CeIPSecPSK},
						{Name: "CE_IPSEC_DEBUG", Value: strconv.FormatBool(cr.Spec.CeIPSecDebug)},
						{Name: "SUBMARINER_HEALTHCHECKENABLED", Value: strconv.FormatBool(healthCheckEnabled)},
						{Name: "SUBMARINER_HEALTHCHECKINTERVAL", Value: strconv.FormatUint(healthCheckInterval, 10)},
						{Name: "SUBMARINER_HEALTHCHECKMAXPACKETLOSSCOUNT", Value: strconv.FormatUint(healthCheckMaxPacketLossCount, 10)},
						{Name: "NODE_NAME", ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "spec.nodeName",
							},
						}},
					},
				},
			},
			// TODO: Use SA submariner-gateway or submariner?
			ServiceAccountName:            "submariner-gateway",
			HostNetwork:                   true,
			TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
			RestartPolicy:                 corev1.RestartPolicyAlways,
			DNSPolicy:                     corev1.DNSClusterFirst,
			// The gateway engine must be able to run on any flagged node, regardless of existing taints
			Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
		},
	}
	if cr.Spec.CeIPSecIKEPort != 0 {
		podTemplate.Spec.Containers[0].Env = append(podTemplate.Spec.Containers[0].Env,
			corev1.EnvVar{Name: "CE_IPSEC_IKEPORT", Value: strconv.Itoa(cr.Spec.CeIPSecIKEPort)})
	}

	if cr.Spec.CeIPSecNATTPort != 0 {
		podTemplate.Spec.Containers[0].Env = append(podTemplate.Spec.Containers[0].Env,
			corev1.EnvVar{Name: "CE_IPSEC_NATTPORT", Value: strconv.Itoa(cr.Spec.CeIPSecNATTPort)})
	}

	podTemplate.Spec.Containers[0].Env = append(podTemplate.Spec.Containers[0].Env,
		corev1.EnvVar{Name: "CE_IPSEC_PREFERREDSERVER", Value: strconv.FormatBool(cr.Spec.CeIPSecPreferredServer)})

	podTemplate.Spec.Containers[0].Env = append(podTemplate.Spec.Containers[0].Env,
		corev1.EnvVar{Name: "CE_IPSEC_FORCEENCAPS", Value: strconv.FormatBool(cr.Spec.CeIPSecForceUDPEncaps)})

	return podTemplate
}

func (r *SubmarinerReconciler) reconcileGatewayDaemonSet(
	instance *v1alpha1.Submariner, reqLogger logr.Logger) (*appsv1.DaemonSet, error) {
	daemonSet, err := helpers.ReconcileDaemonSet(instance, newGatewayDaemonSet(instance), reqLogger, r.client, r.scheme)
	if err != nil {
		return nil, err
	}
	err = metrics.Setup(instance.Namespace, instance, daemonSet.GetLabels(), gatewayMetricsServerPort, r.client, r.config, r.scheme, reqLogger)
	return daemonSet, err
}

func buildGatewayStatusAndUpdateMetrics(gateways *[]submarinerv1.Gateway) []submarinerv1.GatewayStatus {
	var gatewayStatuses = []submarinerv1.GatewayStatus{}

	if gateways != nil {
		recordGateways(len(*gateways))
		// Clear the connections so we don’t remember stale status information
		recordNoConnections()
		for _, gateway := range *gateways {
			gatewayStatuses = append(gatewayStatuses, gateway.Status)
			recordGatewayCreationTime(gateway.Status.LocalEndpoint, gateway.CreationTimestamp.Time)

			for j := range gateway.Status.Connections {
				recordConnection(
					gateway.Status.LocalEndpoint,
					gateway.Status.Connections[j].Endpoint,
					string(gateway.Status.Connections[j].Status),
				)
			}
		}
	} else {
		recordGateways(0)
		recordNoConnections()
	}

	return gatewayStatuses
}

func (r *SubmarinerReconciler) retrieveGateways(ctx context.Context, owner metav1.Object,
	namespace string) (*[]submarinerv1.Gateway, error) {
	foundGateways := &submarinerv1.GatewayList{}
	err := r.client.List(ctx, foundGateways, client.InNamespace(namespace))
	if err != nil && errors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Ensure we’ll get updates
	for i := range foundGateways.Items {
		if err := controllerutil.SetControllerReference(owner, &foundGateways.Items[i], r.scheme); err != nil {
			return nil, err
		}
	}
	return &foundGateways.Items, nil
}
