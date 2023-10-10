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

	"github.com/submariner-io/admiral/pkg/finalizer"
	"github.com/submariner-io/admiral/pkg/names"
	"github.com/submariner-io/admiral/pkg/resource"
	operatorv1alpha1 "github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/uninstall"
	"github.com/submariner-io/submariner-operator/pkg/images"
	opnames "github.com/submariner-io/submariner-operator/pkg/names"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *Reconciler) runComponentCleanup(ctx context.Context, instance *operatorv1alpha1.Submariner) (reconcile.Result, error) {
	if !finalizer.IsPresent(instance, opnames.CleanupFinalizer) {
		return reconcile.Result{}, nil
	}

	if !uninstall.IsSupportedForVersion(instance.Spec.Version) {
		log.Info("Deleting Submariner version does not support uninstall", "version", instance.Spec.Version)
		return reconcile.Result{}, r.removeFinalizer(ctx, instance)
	}

	// This has the side effect of setting the CIDRs in the Submariner instance.
	_, err := r.discoverNetwork(ctx, instance, log)
	if err != nil {
		return reconcile.Result{}, err
	}

	components := []*uninstall.Component{
		{
			Resource:          newDaemonSet(names.GatewayComponent, instance.Namespace),
			UninstallResource: newGatewayDaemonSet(instance, opnames.AppendUninstall(names.GatewayComponent)),
		},
		{
			Resource:          newDaemonSet(names.RouteAgentComponent, instance.Namespace),
			UninstallResource: newRouteAgentDaemonSet(instance, opnames.AppendUninstall(names.RouteAgentComponent)),
		},
		{
			Resource:          newDaemonSet(names.GlobalnetComponent, instance.Namespace),
			UninstallResource: newGlobalnetDaemonSet(instance, opnames.AppendUninstall(names.GlobalnetComponent)),
			CheckInstalled: func() bool {
				return instance.Spec.GlobalCIDR != ""
			},
		},
	}

	uninstallInfo := &uninstall.Info{
		Client:     r.config.ScopedClient,
		Components: components,
		StartTime:  instance.DeletionTimestamp.Time,
		Log:        log,
		GetImageInfo: func(imageName, componentName string) (string, corev1.PullPolicy) {
			return getImagePath(instance, imageName, componentName),
				images.GetPullPolicy(instance.Spec.Version, instance.Spec.ImageOverrides[componentName])
		},
	}

	requeue, timedOut, err := uninstallInfo.Run(ctx)
	if err != nil {
		return reconcile.Result{}, err //nolint:wrapcheck // No need to wrap
	}

	if !timedOut && instance.Spec.ServiceDiscoveryEnabled {
		requeue = r.ensureServiceDiscoveryDeleted(ctx, instance.Namespace) || requeue
	}

	if requeue {
		return reconcile.Result{RequeueAfter: time.Millisecond * 500}, nil
	}

	return reconcile.Result{}, r.removeFinalizer(ctx, instance)
}

func (r *Reconciler) removeFinalizer(ctx context.Context, instance *operatorv1alpha1.Submariner) error {
	return finalizer.Remove[*operatorv1alpha1.Submariner](ctx, resource.ForControllerClient(
		r.config.ScopedClient, instance.Namespace, &operatorv1alpha1.Submariner{}),
		instance, opnames.CleanupFinalizer)
}

func (r *Reconciler) ensureServiceDiscoveryDeleted(ctx context.Context, namespace string) bool {
	err := r.config.ScopedClient.Delete(ctx, newServiceDiscoveryCR(namespace))
	if apierrors.IsNotFound(err) {
		return false
	}

	if err == nil {
		log.Info("Deleted the ServiceDiscovery resource")
	} else {
		log.Error(err, "Error deleting the ServiceDiscovery resource")
	}

	return true
}

func newDaemonSet(name, namespace string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}
