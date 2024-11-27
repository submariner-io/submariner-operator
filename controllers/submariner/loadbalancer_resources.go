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
	configv1 "github.com/openshift/api/config/v1"
	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/names"
	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/apply"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	"github.com/submariner-io/submariner/pkg/port"
	corev1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	loadBalancerName      = "submariner-gateway"
	gatewayStatusLabel    = "gateway.submariner.io/status"
	encapsPortName        = "cable-encaps"
	nattDiscoveryPortName = "natt-discovery"
)

//nolint:wrapcheck // No need to wrap errors here.
func (r *Reconciler) reconcileLoadBalancer(
	ctx context.Context, instance *v1alpha1.Submariner, reqLogger logr.Logger,
) (*corev1.Service, error) {
	platformTypeOCP, err := r.getOCPPlatformType(ctx)
	if err != nil {
		return nil, err
	}

	svc, err := apply.Service(ctx, instance, newLoadBalancerService(instance, platformTypeOCP),
		reqLogger, r.config.ScopedClient, r.config.Scheme)
	if err != nil {
		return nil, err
	}

	// For IBM cloud also needs to annotate the allocated health check node port
	if platformTypeOCP == string(configv1.IBMCloudPlatformType) {
		healthPortStr := strconv.Itoa(int(svc.Spec.HealthCheckNodePort))
		svc.ObjectMeta.Annotations = map[string]string{
			"service.kubernetes.io/ibm-load-balancer-cloud-provider-vpc-health-check-port": healthPortStr,
		}
		svc, err = apply.Service(ctx, instance, svc, reqLogger, r.config.ScopedClient, r.config.Scheme)
	}

	return svc, err
}

func (r *Reconciler) getOCPPlatformType(ctx context.Context) (string, error) {
	clusterInfra := &configv1.Infrastructure{}
	err := r.config.GeneralClient.Get(ctx, types.NamespacedName{Name: "cluster"}, clusterInfra)

	if resource.IsNotFoundErr(err) {
		return "", nil
	}

	if err != nil {
		return "", errors.Wrap(err, "error retrieving cluster Infrastructure resource")
	}

	if clusterInfra.Status.PlatformStatus == nil {
		return string(clusterInfra.Status.Platform), nil //nolint:staticcheck //Purposely using deprecated field for backwards compatibility
	}

	return string(clusterInfra.Status.PlatformStatus.Type), nil
}

func newLoadBalancerService(instance *v1alpha1.Submariner, platformTypeOCP string) *corev1.Service {
	var svcAnnotations map[string]string

	switch platformTypeOCP {
	case string(configv1.AWSPlatformType):
		svcAnnotations = map[string]string{
			"service.beta.kubernetes.io/aws-load-balancer-type": "nlb",
		}
	case string(configv1.IBMCloudPlatformType):
		svcAnnotations = map[string]string{
			"service.kubernetes.io/ibm-load-balancer-cloud-provider-enable-features":           "nlb",
			"service.kubernetes.io/ibm-load-balancer-cloud-provider-ip-type":                   "public",
			"service.kubernetes.io/ibm-load-balancer-cloud-provider-vpc-health-check-protocol": "http",
		}
	default:
		svcAnnotations = map[string]string{}
	}

	return &corev1.Service{
		ObjectMeta: v1meta.ObjectMeta{
			Name:        loadBalancerName,
			Namespace:   instance.Spec.Namespace,
			Annotations: svcAnnotations,
		},
		Spec: corev1.ServiceSpec{
			ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeLocal,
			Type:                  corev1.ServiceTypeLoadBalancer,
			Selector: map[string]string{
				// Traffic is directed to the active gateway
				appLabel:           names.GatewayComponent,
				gatewayStatusLabel: string(submv1.HAStatusActive),
			},
			Ports: []corev1.ServicePort{
				{
					Name:       encapsPortName,
					Port:       toInt32(instance.Spec.CeIPSecNATTPort),
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: toInt32(instance.Spec.CeIPSecNATTPort)},
					Protocol:   corev1.ProtocolUDP,
				},
				{
					Name:       nattDiscoveryPortName,
					Port:       int32(port.NATTDiscovery),
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(port.NATTDiscovery)},
					Protocol:   corev1.ProtocolUDP,
				},
			},
		},
	}
}
