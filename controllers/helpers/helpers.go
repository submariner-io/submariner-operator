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

// nolint:dupl // These functions are similar but not duplicated.
package helpers

import (
	"context"
	goerrors "errors"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/pkg/images"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func ReconcileDaemonSet(owner metav1.Object, daemonSet *appsv1.DaemonSet, reqLogger logr.Logger,
	client controllerClient.Client, scheme *runtime.Scheme) (*appsv1.DaemonSet, error) {
	var err error

	// Set the owner and controller.
	if err := controllerutil.SetControllerReference(owner, daemonSet, scheme); err != nil {
		return nil, err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		toUpdate := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{
			Name:      daemonSet.Name,
			Namespace: daemonSet.Namespace,
			Labels:    map[string]string{},
		}}

		result, err := controllerutil.CreateOrUpdate(context.TODO(), client, toUpdate, func() error {
			toUpdate.Spec = daemonSet.Spec
			for k, v := range daemonSet.Labels {
				toUpdate.Labels[k] = v
			}
			// Set the owner and controller.
			return controllerutil.SetControllerReference(owner, toUpdate, scheme)
		})
		if err != nil {
			if IsImmutableError(err) {
				reqLogger.Info("Re-creating a DaemonSet because it has immutable fields", "DaemonSet.Namespace",
					daemonSet.Namespace, "DaemonSet.Name", daemonSet.Name)

				if err := client.Delete(context.TODO(), toUpdate); err != nil {
					return err
				}

				if err := client.Create(context.TODO(), daemonSet); err != nil {
					return err
				}

				return nil
			}
			return err
		}

		if result == controllerutil.OperationResultCreated {
			reqLogger.Info("Created a new DaemonSet", "DaemonSet.Namespace", daemonSet.Namespace, "DaemonSet.Name", daemonSet.Name)
		} else if result == controllerutil.OperationResultUpdated {
			reqLogger.Info("Updated existing DaemonSet", "DaemonSet.Namespace", daemonSet.Namespace, "DaemonSet.Name", daemonSet.Name)
		}

		return nil
	})

	// Update the status from the server
	if err == nil {
		err = client.Get(context.TODO(), types.NamespacedName{Namespace: daemonSet.Namespace, Name: daemonSet.Name}, daemonSet)
	}

	return daemonSet, errors.WithMessagef(err, "error creating or updating DaemonSet %s/%s", daemonSet.Namespace, daemonSet.Name)
}

func ReconcileDeployment(owner metav1.Object, deployment *appsv1.Deployment, reqLogger logr.Logger,
	client controllerClient.Client, scheme *runtime.Scheme) (*appsv1.Deployment, error) {
	var err error

	// Set the owner and controller
	if err := controllerutil.SetControllerReference(owner, deployment, scheme); err != nil {
		return nil, err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		toUpdate := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
			Labels:    map[string]string{},
		}}

		result, err := controllerutil.CreateOrUpdate(context.TODO(), client, toUpdate, func() error {
			toUpdate.Spec = deployment.Spec
			for k, v := range deployment.Labels {
				toUpdate.Labels[k] = v
			}
			// Set the owner and controller
			return controllerutil.SetControllerReference(owner, toUpdate, scheme)
		})
		if err != nil {
			return err
		}

		if result == controllerutil.OperationResultCreated {
			reqLogger.Info("Created a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		} else if result == controllerutil.OperationResultUpdated {
			reqLogger.Info("Updated existing Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		}

		return nil
	})

	// Update the status from the server
	if err == nil {
		err = client.Get(context.TODO(), types.NamespacedName{Namespace: deployment.Namespace, Name: deployment.Name}, deployment)
	}

	return deployment, errors.WithMessagef(err, "error creating or updating Deployment %s/%s", deployment.Namespace, deployment.Name)
}

func ReconcileConfigMap(owner metav1.Object, configMap *corev1.ConfigMap, reqLogger logr.Logger,
	client controllerClient.Client, scheme *runtime.Scheme) (*corev1.ConfigMap, error) {
	var err error

	// Set the owner and controller
	if err := controllerutil.SetControllerReference(owner, configMap, scheme); err != nil {
		return nil, err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		toUpdate := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Name:      configMap.Name,
			Namespace: configMap.Namespace,
			Labels:    map[string]string{},
		}}

		result, err := controllerutil.CreateOrUpdate(context.TODO(), client, toUpdate, func() error {
			toUpdate.Data = configMap.Data
			for k, v := range configMap.Labels {
				toUpdate.Labels[k] = v
			}
			// Set the owner and controller
			return controllerutil.SetControllerReference(owner, toUpdate, scheme)
		})
		if err != nil {
			return err
		}

		if result == controllerutil.OperationResultCreated {
			reqLogger.Info("Created a new ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
		} else if result == controllerutil.OperationResultUpdated {
			reqLogger.Info("Updated existing ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
		}

		return nil
	})

	// Update the status from the server
	if err == nil {
		err = client.Get(context.TODO(), types.NamespacedName{Namespace: configMap.Namespace, Name: configMap.Name}, configMap)
	}

	return configMap, errors.WithMessagef(err, "error creating or updating ConfigMap %s/%s", configMap.Namespace, configMap.Name)
}

func ReconcileService(owner metav1.Object, service *corev1.Service, reqLogger logr.Logger,
	client controllerClient.Client, scheme *runtime.Scheme) (*corev1.Service, error) {
	var err error

	// Set the owner and controller
	if err := controllerutil.SetControllerReference(owner, service, scheme); err != nil {
		return nil, err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		toUpdate := &corev1.Service{ObjectMeta: metav1.ObjectMeta{
			Name:        service.Name,
			Namespace:   service.Namespace,
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		}}

		result, err := controllerutil.CreateOrUpdate(context.TODO(), client, toUpdate, func() error {
			if toUpdate.Spec.Type == corev1.ServiceTypeClusterIP || toUpdate.Spec.Type == corev1.ServiceTypeLoadBalancer {
				// Make sure we don't lose the ClusterIP, see https://github.com/kubernetes/kubectl/issues/798
				service.Spec.ClusterIP = toUpdate.Spec.ClusterIP
				// We also need to copy the existing healthCheckNodePort which can't change
				service.Spec.HealthCheckNodePort = toUpdate.Spec.HealthCheckNodePort
			}
			toUpdate.Spec = service.Spec
			for k, v := range service.Labels {
				toUpdate.Labels[k] = v
			}
			for k, v := range service.Annotations {
				toUpdate.Annotations[k] = v
			}
			// Set the owner and controller
			return controllerutil.SetControllerReference(owner, toUpdate, scheme)
		})
		if err != nil {
			return err
		}

		if result == controllerutil.OperationResultCreated {
			reqLogger.Info("Created a new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		} else if result == controllerutil.OperationResultUpdated {
			reqLogger.Info("Updated existing Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		}

		return nil
	})

	// Update the status from the server
	if err == nil {
		err = client.Get(context.TODO(), types.NamespacedName{Namespace: service.Namespace, Name: service.Name}, service)
	}

	return service, errors.WithMessagef(err, "error creating or updating Service %s/%s", service.Namespace, service.Name)
}

func GetPullPolicy(version, override string) corev1.PullPolicy {
	if len(override) > 0 {
		tag := strings.Split(override, ":")[1]
		return images.GetPullPolicy(tag)
	}
	return images.GetPullPolicy(version)
}

func IsImmutableError(err error) bool {
	if !apierrors.IsInvalid(err) {
		return false
	}

	if status := apierrors.APIStatus(nil); goerrors.As(err, &status) {
		if strings.Contains(status.Status().Message, "immutable") {
			return true
		}
	}

	return false
}
