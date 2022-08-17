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
	"fmt"

	"github.com/pkg/errors"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monclientv1 "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var ErrServiceMonitorNotPresent = fmt.Errorf("no ServiceMonitor registered with the API")

const openshiftMonitoringNS = "openshift-monitoring"

type ServiceMonitorUpdater func(*monitoringv1.ServiceMonitor) error

// CreateServiceMonitors creates ServiceMonitors objects based on an array of Service objects.
// If CR ServiceMonitor is not registered in the Cluster it will not attempt at creating resources.
func CreateServiceMonitors(config *rest.Config, ns string, services []*v1.Service) ([]*monitoringv1.ServiceMonitor, error) {
	// check if we can even create ServiceMonitors
	exists, err := hasServiceMonitor(config)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrServiceMonitorNotPresent
	}

	// On OpenShift, we need to create the service monitors in the OpenShift monitoring namespace, not the
	// services; we need our own clientset rather than the manager's since the latter hasn't started yet
	// (so its caching infrastructure isn't available, and reads fail)
	cs, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "error getting kube client")
	}

	if _, err := cs.CoreV1().Namespaces().Get(context.TODO(), openshiftMonitoringNS, metav1.GetOptions{}); err == nil {
		ns = openshiftMonitoringNS
	} else if !apierrors.IsNotFound(err) {
		log.Error(err, "Error checking for the OpenShift monitoring namespace")
	}

	serviceMonitors := make([]*monitoringv1.ServiceMonitor, len(services))
	mclient := monclientv1.NewForConfigOrDie(config)

	for i, s := range services {
		if s == nil {
			continue
		}

		sm := GenerateServiceMonitor(ns, s)

		smc, err := mclient.ServiceMonitors(ns).Create(context.TODO(), sm, metav1.CreateOptions{})
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
	boolTrue := true

	// Owner references only work inside the same namespace
	ownerReferences := []metav1.OwnerReference{}
	namespaceSelector := monitoringv1.NamespaceSelector{}

	if ns == s.ObjectMeta.Namespace {
		ownerReferences = []metav1.OwnerReference{
			{
				APIVersion:         "v1",
				BlockOwnerDeletion: &boolTrue,
				Controller:         &boolTrue,
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
	dc := discovery.NewDiscoveryClientForConfigOrDie(config)
	apiVersion := "monitoring.coreos.com/v1"
	kind := "ServiceMonitor"

	return resourceExists(dc, apiVersion, kind) // nolint:wrapcheck // No need to wrap here
}

// resourceExists returns true if the given resource kind exists in the given api groupversion.
func resourceExists(dc discovery.DiscoveryInterface, apiGroupVersion, kind string) (bool, error) {
	_, apiLists, err := dc.ServerGroupsAndResources()
	if err != nil {
		return false, err
	}

	for _, apiList := range apiLists {
		if apiList.GroupVersion == apiGroupVersion {
			for _, r := range apiList.APIResources {
				if r.Kind == kind {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
