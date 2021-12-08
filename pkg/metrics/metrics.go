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
	"errors"
	"fmt"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/admiral/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("metrics")

const (
	// OperatorPortName defines the default operator metrics port name used in the metrics Service.
	OperatorPortName = "http-metrics"
	// CRPortName defines the custom resource specific metrics' port name used in the metrics Service.
	CRPortName = "cr-metrics"
)

// CreateMetricsService creates a Kubernetes Service to expose the passed metrics
// port(s) with the given name(s).
func CreateMetricsService(ctx context.Context, cfg *rest.Config, servicePorts []v1.ServicePort) (*v1.Service, bool, error) {
	if len(servicePorts) < 1 {
		return nil, false, fmt.Errorf("failed to create metrics Serice; service ports were empty")
	}
	client, err := crclient.New(cfg, crclient.Options{})
	if err != nil {
		return nil, false, fmt.Errorf("failed to create new client: %w", err)
	}
	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create clientset: %w", err)
	}
	s, err := initOperatorService(ctx, client, servicePorts)
	if err != nil {
		if errors.Is(err, k8sutil.ErrNoNamespace) || errors.Is(err, k8sutil.ErrRunLocal) {
			log.Info("Skipping metrics Service creation; not running in a cluster.")
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to initialize service object for metrics: %w", err)
	}
	_, err = util.CreateOrUpdate(ctx, resource.ForService(clientSet, s.Namespace),
		s, func(existing runtime.Object) (runtime.Object, error) {
			existingService := existing.(*v1.Service)
			if existingService.Spec.Type == v1.ServiceTypeClusterIP {
				s.Spec.ClusterIP = existingService.Spec.ClusterIP
			}
			return s, nil
		})
	if err != nil {
		return nil, false, err
	}

	s, err = clientSet.CoreV1().Services(s.Namespace).Get(ctx, s.Name, metav1.GetOptions{})
	return s, true, err
}

// initOperatorService returns the static service which exposes specified port(s).
func initOperatorService(ctx context.Context, client crclient.Client, sp []v1.ServicePort) (*v1.Service, error) {
	operatorName, err := k8sutil.GetOperatorName()
	if err != nil {
		return nil, err
	}
	namespace, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return nil, err
	}
	label := map[string]string{"name": operatorName}

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-metrics", operatorName),
			Namespace: namespace,
			Labels:    label,
		},
		Spec: v1.ServiceSpec{
			Ports:    sp,
			Selector: label,
		},
	}

	ownRef, err := getPodOwnerRef(ctx, client, namespace)
	if err != nil {
		return nil, err
	}
	service.SetOwnerReferences([]metav1.OwnerReference{*ownRef})

	return service, nil
}

func getPodOwnerRef(ctx context.Context, client crclient.Client, ns string) (*metav1.OwnerReference, error) {
	// Get current Pod the operator is running in
	pod, err := k8sutil.GetPod(ctx, client, ns)
	if err != nil {
		return nil, err
	}
	podOwnerRefs := metav1.NewControllerRef(pod, pod.GroupVersionKind())
	// Get Owner that the Pod belongs to
	ownerRef := metav1.GetControllerOf(pod)
	finalOwnerRef, found, err := findFinalOwnerRef(ctx, client, ns, ownerRef)
	if err != nil {
		return nil, err
	}

	if found {
		return finalOwnerRef, nil
	}

	// Default to returning Pod as the Owner
	return podOwnerRefs, nil
}

// findFinalOwnerRef tries to locate the final controller/owner based on the owner reference provided.
func findFinalOwnerRef(ctx context.Context, client crclient.Client, ns string,
	ownerRef *metav1.OwnerReference) (*metav1.OwnerReference, bool, error) {
	if ownerRef == nil {
		return nil, false, nil
	}

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(ownerRef.APIVersion)
	obj.SetKind(ownerRef.Kind)
	err := client.Get(ctx, types.NamespacedName{Namespace: ns, Name: ownerRef.Name}, obj)
	if err != nil {
		return nil, false, err
	}

	newOwnerRef := metav1.GetControllerOf(obj)
	if newOwnerRef != nil {
		return findFinalOwnerRef(ctx, client, ns, newOwnerRef)
	}

	log.V(1).Info("Pods owner found", "Kind", ownerRef.Kind, "Name",
		ownerRef.Name, "Namespace", ns)

	return ownerRef, true, nil
}
