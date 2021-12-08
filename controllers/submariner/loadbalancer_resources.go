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
	"github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/helpers"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	corev1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	loadBalancerName      = "submariner-gateway"
	gatewayStatusLabel    = "gateway.submariner.io/status"
	encapsPortName        = "cable-encaps"
	nattDiscoveryPortName = "natt-discovery"
)

func (r *SubmarinerReconciler) reconcileLoadBalancer(
	instance *v1alpha1.Submariner, reqLogger logr.Logger) (*corev1.Service, error) {
	service, err := helpers.ReconcileService(instance, newLoadBalancerService(instance), reqLogger, r.client, r.scheme)
	if err != nil {
		return nil, err
	}
	return service, err
}

func newLoadBalancerService(instance *v1alpha1.Submariner) *corev1.Service {
	nattPort, _ := strconv.ParseInt(submv1.DefaultNATTDiscoveryPort, 10, 32)

	return &corev1.Service{
		ObjectMeta: v1meta.ObjectMeta{
			Name:      loadBalancerName,
			Namespace: instance.Spec.Namespace,
			Annotations: map[string]string{
				// AWS requires nlb Load Balancer for UDP
				"service.beta.kubernetes.io/aws-load-balancer-type": "nlb",
			},
		},
		Spec: corev1.ServiceSpec{
			ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeLocal,
			Type:                  corev1.ServiceTypeLoadBalancer,
			Selector: map[string]string{
				// Traffic is directed to the active gateway
				appLabel:           appGatewayLabel,
				gatewayStatusLabel: string(submv1.HAStatusActive),
			},
			Ports: []corev1.ServicePort{
				{
					Name:       encapsPortName,
					Port:       int32(instance.Spec.CeIPSecNATTPort),
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(instance.Spec.CeIPSecNATTPort)},
					Protocol:   corev1.ProtocolUDP,
				},
				{
					Name:       nattDiscoveryPortName,
					Port:       int32(nattPort),
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(nattPort)},
					Protocol:   corev1.ProtocolUDP,
				},
			},
		},
	}
}
