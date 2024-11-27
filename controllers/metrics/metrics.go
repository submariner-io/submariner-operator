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

package metrics

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	"github.com/submariner-io/submariner-operator/controllers/apply"
	"github.com/submariner-io/submariner-operator/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceInfo struct {
	Name            string
	Namespace       string
	ApplicationKey  string
	ApplicationName string
	Owner           metav1.Object
	Port            int32
}

func Setup(ctx context.Context, client controllerClient.Client, config *rest.Config, scheme *runtime.Scheme,
	serviceInfo *ServiceInfo, reqLogger logr.Logger,
) error {
	metricsService, err := apply.Service(ctx, serviceInfo.Owner,
		newMetricsService(serviceInfo.Name, serviceInfo.Namespace, serviceInfo.ApplicationKey,
			serviceInfo.ApplicationName, serviceInfo.Port), reqLogger, client, scheme)
	if err != nil {
		return err //nolint:wrapcheck // No need to wrap here
	}

	if config != nil {
		services := []*corev1.Service{metricsService}

		_, err = metrics.CreateServiceMonitors(ctx, config, serviceInfo.Namespace, services)
		if err != nil {
			// If this operator is deployed to a cluster without the prometheus-operator running, it will return
			// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
			if errors.Is(err, metrics.ErrServiceMonitorNotPresent) {
				reqLogger.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
			} else if !k8serrors.IsAlreadyExists(err) {
				return err //nolint:wrapcheck // No need to wrap here
			}
		}
	}

	return nil
}

// newMetricsService populates a Service providing access to metrics for the given application.
// The Service is named after the application name, suffixed with "-metrics".
func newMetricsService(name, namespace, appKey, appName string, port int32) *corev1.Service {
	labels := map[string]string{
		appKey: appName,
	}

	if name == "" {
		name = appName
	}

	servicePorts := []corev1.ServicePort{
		{Port: port, Name: "metrics", Protocol: corev1.ProtocolTCP, TargetPort: intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: port,
		}},
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Namespace: namespace,
			Name:      name + "-metrics",
		},
		Spec: corev1.ServiceSpec{
			Ports:    servicePorts,
			Selector: labels,
		},
	}

	return service
}
