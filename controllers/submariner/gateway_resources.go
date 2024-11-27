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
	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/names"
	"github.com/submariner-io/admiral/pkg/syncer/broker"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/apply"
	"github.com/submariner-io/submariner-operator/controllers/metrics"
	"github.com/submariner-io/submariner-operator/pkg/httpproxy"
	"github.com/submariner-io/submariner-operator/pkg/images"
	opnames "github.com/submariner-io/submariner-operator/pkg/names"
	submarinerv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	"github.com/submariner-io/submariner/pkg/port"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	appLabel = "app"
)

func newGatewayDaemonSet(cr *v1alpha1.Submariner, name string) *appsv1.DaemonSet {
	maxUnavailable := intstr.FromInt(1)
	podSelectorLabels := map[string]string{appLabel: name}

	deployment := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				appLabel:    name,
				"component": "gateway",
			},
			Namespace: cr.Namespace,
			Name:      name,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: podSelectorLabels},
			Template: newGatewayPodTemplate(cr, name, podSelectorLabels),
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &maxUnavailable,
				},
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
			},
			RevisionHistoryLimit: ptr.To(int32(5)),
		},
	}

	return deployment
}

// newGatewayPodTemplate returns a submariner pod with the same fields as the cr.
func newGatewayPodTemplate(cr *v1alpha1.Submariner, name string, podSelectorLabels map[string]string) corev1.PodTemplateSpec {
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

	volumeMounts := []corev1.VolumeMount{
		{Name: "ipsecd", MountPath: "/etc/ipsec.d", ReadOnly: false},
		{Name: "ipsecnss", MountPath: "/var/lib/ipsec/nss", ReadOnly: false},
		{Name: "libmodules", MountPath: "/lib/modules", ReadOnly: true},
	}
	volumes := []corev1.Volume{
		{Name: "ipsecd", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{Name: "ipsecnss", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{Name: "libmodules", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
			Path: "/lib/modules", Type: ptr.To(corev1.HostPathDirectoryOrCreate),
		}}},
	}

	if cr.Spec.BrokerK8sSecret != "" {
		// We've got a secret, mount it where the syncer expects it
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "brokersecret",
			MountPath: broker.SecretPath(cr.Spec.BrokerK8sSecret),
			ReadOnly:  true,
		})

		volumes = append(volumes, corev1.Volume{
			Name:         "brokersecret",
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: cr.Spec.BrokerK8sSecret}},
		})
	}

	if cr.Spec.CeIPSecPSKSecret != "" {
		// We've got a PSK secret, mount it where the gateway expects it
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "psksecret",
			MountPath: "/var/run/secrets/submariner.io/" + cr.Spec.CeIPSecPSKSecret,
			ReadOnly:  true,
		})

		volumes = append(volumes, corev1.Volume{
			Name:         "psksecret",
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: cr.Spec.CeIPSecPSKSecret}},
		})
	}

	securityContext := &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Add:  []corev1.Capability{"net_admin"},
			Drop: []corev1.Capability{"all"},
		},
		// The gateway needs to be privileged so it can write to /proc/sys
		AllowPrivilegeEscalation: ptr.To(true),
		Privileged:               ptr.To(true),
		RunAsNonRoot:             ptr.To(false),
		// We need to be able to update /var/lib/alternatives (for iptables)
		ReadOnlyRootFilesystem: ptr.To(false),
	}

	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: podSelectorLabels,
		},
		Spec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: podSelectorLabels,
						},
						TopologyKey: "kubernetes.io/hostname",
					}},
				},
			},
			NodeSelector: map[string]string{"submariner.io/gateway": "true"},
			// Wait for the node to be ready before starting the gateway.
			InitContainers: []corev1.Container{
				{
					Name:            name + "-init",
					Image:           getImagePath(cr, opnames.GatewayImage, names.GatewayComponent),
					ImagePullPolicy: images.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.GatewayComponent]),
					SecurityContext: securityContext,
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
					Image:           getImagePath(cr, opnames.GatewayImage, names.GatewayComponent),
					ImagePullPolicy: images.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.GatewayComponent]),
					SecurityContext: securityContext,
					Ports: []corev1.ContainerPort{
						{
							Name:          encapsPortName,
							HostPort:      toInt32(cr.Spec.CeIPSecNATTPort),
							ContainerPort: toInt32(cr.Spec.CeIPSecNATTPort),
							Protocol:      corev1.ProtocolUDP,
						},
						{
							Name:          nattDiscoveryPortName,
							HostPort:      int32(port.NATTDiscovery),
							ContainerPort: int32(port.NATTDiscovery),
							Protocol:      corev1.ProtocolUDP,
						},
					},
					Env: httpproxy.AddEnvVars([]corev1.EnvVar{
						{Name: "SUBMARINER_NAMESPACE", Value: cr.Spec.Namespace},
						{Name: "SUBMARINER_CLUSTERCIDR", Value: cr.Status.ClusterCIDR},
						{Name: "SUBMARINER_SERVICECIDR", Value: cr.Status.ServiceCIDR},
						{Name: "SUBMARINER_GLOBALCIDR", Value: cr.Spec.GlobalCIDR},
						{Name: "SUBMARINER_CLUSTERID", Value: cr.Spec.ClusterID},
						{Name: "SUBMARINER_COLORCODES", Value: cr.Spec.ColorCodes},
						{Name: "SUBMARINER_DEBUG", Value: strconv.FormatBool(cr.Spec.Debug)},
						{Name: "SUBMARINER_NATENABLED", Value: strconv.FormatBool(cr.Spec.NatEnabled)},
						{Name: "AIR_GAPPED_DEPLOYMENT", Value: strconv.FormatBool(cr.Spec.AirGappedDeployment)},
						{Name: "SUBMARINER_BROKER", Value: cr.Spec.Broker},
						{Name: "SUBMARINER_CABLEDRIVER", Value: cr.Spec.CableDriver},
						{Name: broker.EnvironmentVariable("ApiServer"), Value: cr.Spec.BrokerK8sApiServer},
						{Name: broker.EnvironmentVariable("ApiServerToken"), Value: cr.Spec.BrokerK8sApiServerToken},
						{Name: broker.EnvironmentVariable("RemoteNamespace"), Value: cr.Spec.BrokerK8sRemoteNamespace},
						{Name: broker.EnvironmentVariable("CA"), Value: cr.Spec.BrokerK8sCA},
						{Name: broker.EnvironmentVariable("Insecure"), Value: strconv.FormatBool(cr.Spec.BrokerK8sInsecure)},
						{Name: broker.EnvironmentVariable("Secret"), Value: cr.Spec.BrokerK8sSecret},
						{Name: "CE_IPSEC_PSK", Value: cr.Spec.CeIPSecPSK},
						{Name: "CE_IPSEC_PSKSECRET", Value: cr.Spec.CeIPSecPSKSecret},
						{Name: "CE_IPSEC_DEBUG", Value: strconv.FormatBool(cr.Spec.CeIPSecDebug)},
						{Name: "SUBMARINER_HEALTHCHECKENABLED", Value: strconv.FormatBool(healthCheckEnabled)},
						{Name: "SUBMARINER_HEALTHCHECKINTERVAL", Value: strconv.FormatUint(healthCheckInterval, 10)},
						{Name: "SUBMARINER_HEALTHCHECKMAXPACKETLOSSCOUNT", Value: strconv.FormatUint(healthCheckMaxPacketLossCount, 10)},
						{Name: "SUBMARINER_METRICSPORT", Value: gatewayMetricsServerPort},
						{Name: "SUBMARINER_HALT_ON_CERT_ERROR", Value: strconv.FormatBool(cr.Spec.HaltOnCertificateError)},
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
					}),
					VolumeMounts: volumeMounts,
				},
			},
			ServiceAccountName:            names.GatewayComponent,
			HostNetwork:                   true,
			DNSPolicy:                     corev1.DNSClusterFirstWithHostNet,
			TerminationGracePeriodSeconds: ptr.To(int64(1)),
			RestartPolicy:                 corev1.RestartPolicyAlways,
			// The gateway engine must be able to run on any flagged node, regardless of existing taints
			Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
			Volumes:     volumes,
		},
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

//nolint:wrapcheck // No need to wrap errors here.
func (r *Reconciler) reconcileGatewayDaemonSet(
	ctx context.Context, instance *v1alpha1.Submariner, reqLogger logr.Logger,
) (*appsv1.DaemonSet, error) {
	daemonSet, err := apply.DaemonSet(ctx, instance, newGatewayDaemonSet(instance, names.GatewayComponent),
		reqLogger, r.config.ScopedClient, r.config.Scheme)
	if err != nil {
		return nil, err
	}

	err = metrics.Setup(ctx, r.config.ScopedClient, r.config.RestConfig, r.config.Scheme,
		&metrics.ServiceInfo{
			Name:            names.GatewayComponent,
			Namespace:       instance.Namespace,
			ApplicationKey:  "app",
			ApplicationName: names.MetricsProxyComponent,
			Owner:           instance,
			Port:            gatewayMetricsServicePort,
		}, reqLogger)

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

func (r *Reconciler) retrieveGateways(ctx context.Context, owner metav1.Object,
	namespace string,
) ([]submarinerv1.Gateway, error) {
	foundGateways := &submarinerv1.GatewayList{}

	err := r.config.ScopedClient.List(ctx, foundGateways, client.InNamespace(namespace))
	if err != nil && apierrors.IsNotFound(err) {
		return []submarinerv1.Gateway{}, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "error listing Gateway resource")
	}

	// Ensure we’ll get updates
	for i := range foundGateways.Items {
		if err := controllerutil.SetControllerReference(owner, &foundGateways.Items[i], r.config.Scheme); err != nil {
			return nil, errors.Wrap(err, "error setting owner ref for Gateway")
		}
	}

	return foundGateways.Items, nil
}

func toInt32(from int) int32 {
	return int32(from) //nolint:gosec // Need to ignore reported integer overflow conversion
}
