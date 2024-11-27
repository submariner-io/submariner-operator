// SPDX-License-Identifier: Apache-2.0
//
// Copyright Contributors to the Submariner project.
// Copyright 2018 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"context"

	"github.com/pkg/errors"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monclientv1 "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	"github.com/submariner-io/admiral/pkg/resource"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

var ErrServiceMonitorNotPresent = errors.New("no ServiceMonitor registered with the API")

const openshiftMonitoringNS = "openshift-monitoring"

type ServiceMonitorUpdater func(*monitoringv1.ServiceMonitor) error

// CreateServiceMonitors creates ServiceMonitors objects based on an array of Service objects.
// If CR ServiceMonitor is not registered in the Cluster it will not attempt at creating resources.
func CreateServiceMonitors(ctx context.Context, config *rest.Config, ns string, services []*v1.Service,
) ([]*monitoringv1.ServiceMonitor, error) {
	// check if we can even create ServiceMonitors
	exists, err := hasServiceMonitor(config)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrServiceMonitorNotPresent
	}

	serviceMonitors := make([]*monitoringv1.ServiceMonitor, len(services))
	mclient := monclientv1.NewForConfigOrDie(config)

	for i, s := range services {
		if s == nil {
			continue
		}

		// On OpenShift, we need to create the service monitors in the OpenShift monitoring namespace, not the
		// service's. If that namespace doesn't exist then create in the provided namespace.
		smc, err := mclient.ServiceMonitors(openshiftMonitoringNS).Create(ctx, GenerateServiceMonitor(openshiftMonitoringNS, s),
			metav1.CreateOptions{})

		if resource.IsMissingNamespaceErr(err) {
			smc, err = mclient.ServiceMonitors(ns).Create(ctx, GenerateServiceMonitor(ns, s), metav1.CreateOptions{})
		}

		if err != nil {
			return nil, errors.Wrap(err, "error creating ServiceMonitor")
		}

		serviceMonitors[i] = smc
	}

	return serviceMonitors, nil
}

// GenerateServiceMonitor generates a prometheus-operator ServiceMonitor object
// based on the passed Service object.
func GenerateServiceMonitor(ns string, s *v1.Service) *monitoringv1.ServiceMonitor {
	labels := make(map[string]string)
	for k, v := range s.ObjectMeta.Labels {
		labels[k] = v
	}

	endpoints := populateEndpointsFromServicePorts(s)

	// Owner references only work inside the same namespace
	ownerReferences := []metav1.OwnerReference{}
	namespaceSelector := monitoringv1.NamespaceSelector{}

	if ns == s.ObjectMeta.Namespace {
		ownerReferences = []metav1.OwnerReference{
			{
				APIVersion:         "v1",
				BlockOwnerDeletion: ptr.To(true),
				Controller:         ptr.To(true),
				Kind:               "Service",
				Name:               s.Name,
				UID:                s.UID,
			},
		}
	} else {
		namespaceSelector = monitoringv1.NamespaceSelector{
			MatchNames: []string{s.ObjectMeta.Namespace},
		}
	}

	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:            s.ObjectMeta.Name,
			Namespace:       ns,
			Labels:          labels,
			OwnerReferences: ownerReferences,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			NamespaceSelector: namespaceSelector,
			Endpoints:         endpoints,
		},
	}
}

func populateEndpointsFromServicePorts(s *v1.Service) []monitoringv1.Endpoint {
	endpoints := make([]monitoringv1.Endpoint, len(s.Spec.Ports))
	for i := range s.Spec.Ports {
		endpoints[i] = monitoringv1.Endpoint{Port: s.Spec.Ports[i].Name}
	}

	return endpoints
}

// hasServiceMonitor checks if ServiceMonitor is registered in the cluster.
func hasServiceMonitor(config *rest.Config) (bool, error) {
	apiVersion := "monitoring.coreos.com/v1"
	kind := "ServiceMonitor"

	_, apiLists, err := discovery.NewDiscoveryClientForConfigOrDie(config).ServerGroupsAndResources()
	if err != nil {
		return false, err //nolint:wrapcheck // No need to wrap here
	}

	for _, apiList := range apiLists {
		if apiList.GroupVersion == apiVersion {
			for i := range apiList.APIResources {
				if apiList.APIResources[i].Kind == kind {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
