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
	"github.com/submariner-io/admiral/pkg/syncer/broker"
	"github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/helpers"
	"github.com/submariner-io/submariner-operator/controllers/metrics"
	"github.com/submariner-io/submariner-operator/pkg/names"
	submarinerv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

const (
	appLabel        = "app"
	appGatewayLabel = "submariner-gateway"
)

// newGatewayPodTemplate returns a submariner pod with the same fields as the cr.
func newGatewayPodTemplate(cr *v1alpha1.Submariner) corev1.PodTemplateSpec {
	labels := map[string]string{
		appLabel: appGatewayLabel,
	}

	// Create privileged security context for Gateway pod
	// FIXME: Seems like these have to be a var, so can pass pointer to bool var to SecurityContext. Cleaner option?
	// The gateway needs to be privileged so it can write to /proc/sys
	privileged := true
	allowPrivilegeEscalation := true
	runAsNonRoot := false
	// We need to be able to update /var/lib/alternatives (for iptables)
	readOnlyRootFilesystem := false

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

	nattPort, _ := strconv.ParseInt(submarinerv1.DefaultNATTDiscoveryPort, 10, 32)

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
					SecurityContext: &corev1.SecurityContext{
						Capabilities: &corev1.Capabilities{
							Add:  []corev1.Capability{"net_admin"},
							Drop: []corev1.Capability{"all"},
						},
						AllowPrivilegeEscalation: &allowPrivilegeEscalation,
						Privileged:               &privileged,
						ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
						RunAsNonRoot:             &runAsNonRoot,
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          encapsPortName,
							HostPort:      int32(cr.Spec.CeIPSecNATTPort),
							ContainerPort: int32(cr.Spec.CeIPSecNATTPort),
							Protocol:      corev1.ProtocolUDP,
						},
						{
							Name:          nattDiscoveryPortName,
							HostPort:      int32(nattPort),
							ContainerPort: int32(nattPort),
							Protocol:      corev1.ProtocolUDP,
						},
					},
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
						{Name: broker.EnvironmentVariable("ApiServer"), Value: cr.Spec.BrokerK8sApiServer},
						{Name: broker.EnvironmentVariable("ApiServerToken"), Value: cr.Spec.BrokerK8sApiServerToken},
						{Name: broker.EnvironmentVariable("RemoteNamespace"), Value: cr.Spec.BrokerK8sRemoteNamespace},
						{Name: broker.EnvironmentVariable("CA"), Value: cr.Spec.BrokerK8sCA},
						{Name: broker.EnvironmentVariable("Insecure"), Value: strconv.FormatBool(cr.Spec.BrokerK8sInsecure)},
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
						{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "metadata.name",
							},
						}},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "ipsecd", MountPath: "/etc/ipsec.d", ReadOnly: false},
						{Name: "ipsecnss", MountPath: "/var/lib/ipsec/nss", ReadOnly: false},
						{Name: "libmodules", MountPath: "/lib/modules", ReadOnly: true},
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
			Volumes: []corev1.Volume{
				{Name: "ipsecd", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{Name: "ipsecnss", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{Name: "libmodules", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/lib/modules"}}},
			},
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
		corev1.EnvVar{Name: "CE_IPSEC_PREFERREDSERVER", Value: strconv.FormatBool(cr.Spec.CeIPSecPreferredServer ||
			cr.Spec.LoadBalancerEnabled)},
		corev1.EnvVar{Name: "CE_IPSEC_FORCEENCAPS", Value: strconv.FormatBool(cr.Spec.CeIPSecForceUDPEncaps)})

	if cr.Spec.LoadBalancerEnabled {
		podTemplate.Spec.Containers[0].Env = append(podTemplate.Spec.Containers[0].Env,
			corev1.EnvVar{Name: "SUBMARINER_PUBLICIP", Value: "lb:" + loadBalancerName})
	}

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

func buildGatewayStatusAndUpdateMetrics(gateways []submarinerv1.Gateway) []submarinerv1.GatewayStatus {
	gatewayStatuses := []submarinerv1.GatewayStatus{}

	nGateways := len(gateways)
	if nGateways > 0 {
		recordGateways(nGateways)
		// Clear the connections so we don’t remember stale status information
		recordNoConnections()
		for i := range gateways {
			gateway := &gateways[i]
			gatewayStatuses = append(gatewayStatuses, gateway.Status)
			recordGatewayCreationTime(&gateway.Status.LocalEndpoint, gateway.CreationTimestamp.Time)

			for j := range gateway.Status.Connections {
				recordConnection(
					&gateway.Status.LocalEndpoint,
					&gateway.Status.Connections[j].Endpoint,
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
	namespace string) ([]submarinerv1.Gateway, error) {
	foundGateways := &submarinerv1.GatewayList{}
	err := r.client.List(ctx, foundGateways, client.InNamespace(namespace))
	if err != nil && errors.IsNotFound(err) {
		return []submarinerv1.Gateway{}, nil
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
	return foundGateways.Items, nil
}
