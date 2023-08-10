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

	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner/pkg/cni"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const NetworkPluginSyncerComponent = "submariner-networkplugin-syncer"

//nolint:wrapcheck // No need to wrap errors here.
func (r *Reconciler) removeNetworkPluginSyncerDeployment(ctx context.Context, instance *v1alpha1.Submariner) error {
	if instance.Status.NetworkPlugin != cni.OVNKubernetes || r.networkPluginSyncerRemoved {
		return nil
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

	return deleteAll(
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
