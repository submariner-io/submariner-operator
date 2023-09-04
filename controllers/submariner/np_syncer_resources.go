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
	"time"

	"github.com/submariner-io/admiral/pkg/names"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/uninstall"
	"github.com/submariner-io/submariner/pkg/cni"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const NetworkPluginSyncerComponent = "submariner-networkplugin-syncer"

//nolint:wrapcheck // No need to wrap errors here.
func (r *Reconciler) removeNetworkPluginSyncerDeployment(ctx context.Context, instance *v1alpha1.Submariner) (reconcile.Result, error) {
	if instance.Status.NetworkPlugin != cni.OVNKubernetes || r.networkPluginSyncerRemoved {
		return reconcile.Result{}, nil
	}

	if !r.networkPluginSyncerUninstalled {
		requeue, err := r.uninstallNetworkPluginSyncerDeployment(ctx, instance)

		if err != nil && !apierrors.IsNotFound(err) {
			return reconcile.Result{}, err
		}

		if requeue {
			return reconcile.Result{RequeueAfter: time.Millisecond * 500}, err
		}

		r.networkPluginSyncerUninstalled = true
	}

	r.networkPluginSyncerRemoved = true

	deleteAll := func(objs ...client.Object) error {
		for _, obj := range objs {
			obj.SetName(NetworkPluginSyncerComponent)
			obj.SetNamespace(instance.Namespace)

			err := r.config.ScopedClient.Delete(ctx, obj)
			if err != nil && !apierrors.IsNotFound(err) {
				r.networkPluginSyncerRemoved = false
				return err
			}
		}

		return nil
	}

	return reconcile.Result{}, deleteAll(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: instance.Namespace,
				Name:      NetworkPluginSyncerComponent,
			},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: instance.Namespace,
				Name:      NetworkPluginSyncerComponent,
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: instance.Namespace,
				Name:      NetworkPluginSyncerComponent,
			},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: instance.Namespace,
				Name:      "ocp-submariner-networkplugin-syncer",
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: instance.Namespace,
				Name:      "ocp-submariner-networkplugin-syncer",
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: instance.Namespace,
				Name:      NetworkPluginSyncerComponent,
			},
		},
	)
}

//nolint:wrapcheck // No need to wrap errors here.
func (r *Reconciler) uninstallNetworkPluginSyncerDeployment(ctx context.Context, instance *v1alpha1.Submariner) (bool, error) {
	if r.uninstallNPSyncerDeployment != nil {
		return false, nil
	}

	npDeployment := &appsv1.Deployment{}

	err := r.config.ScopedClient.Get(ctx, types.NamespacedName{
		Namespace: instance.Namespace, Name: names.NetworkPluginSyncerComponent,
	}, npDeployment)
	if err != nil {
		return false, err
	}

	npDeployment = networkPluginSyncerUninstallDeployment(npDeployment)
	r.uninstallNPSyncerDeployment = npDeployment

	component := []*uninstall.Component{
		{
			Resource:          newDeployment(names.NetworkPluginSyncerComponent, instance.Namespace),
			UninstallResource: npDeployment,
			CheckInstalled:    func() bool { return true },
		},
	}

	uninstallInfo := &uninstall.Info{
		Client:     r.config.ScopedClient,
		Components: component,
		StartTime:  time.Now(),
		Log:        log,
		GetImageInfo: func(imageName, componentName string) (string, corev1.PullPolicy) {
			container := npDeployment.Spec.Template.Spec.Containers[0]
			return container.Image, container.ImagePullPolicy
		},
	}

	requeue, _, err := uninstallInfo.Run(ctx)
	if err != nil {
		return false, err
	}

	if requeue {
		return true, nil
	}

	return false, nil
}

func networkPluginSyncerUninstallDeployment(oldDeployment *appsv1.Deployment) *appsv1.Deployment {
	name := oldDeployment.Name + "-uninstall"
	labels := map[string]string{
		"app":       name,
		"component": "networkplugin-syncer",
	}
	networkPluginSyncerDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: oldDeployment.Namespace,
			Name:      name,
			Labels:    labels,
		},
	}
	networkPluginSyncerDeployment.Spec = oldDeployment.Spec
	networkPluginSyncerDeployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{
		"app": name,
	}}

	networkPluginSyncerDeployment.Spec.Template.SetLabels(labels)
	networkPluginSyncerDeployment.Spec.Strategy = appsv1.DeploymentStrategy{
		Type: appsv1.RecreateDeploymentStrategyType,
	}

	return networkPluginSyncerDeployment
}

func newDeployment(name, namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}
