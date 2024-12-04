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

package servicediscovery

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/finalizer"
	"github.com/submariner-io/admiral/pkg/names"
	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/admiral/pkg/util"
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

func (r *Reconciler) doCleanup(ctx context.Context, instance *operatorv1alpha1.ServiceDiscovery) (reconcile.Result, error) {
	if !finalizer.IsPresent(instance, opnames.CleanupFinalizer) {
		return reconcile.Result{}, nil
	}

	if !uninstall.IsSupportedForVersion(instance.Spec.Version) {
		log.Info("Deleting ServiceDiscovery version does not support uninstall", "version", instance.Spec.Version)
		return reconcile.Result{}, r.removeFinalizer(ctx, instance)
	}

	var err error

	if instance.Spec.CoreDNSCustomConfig != nil && instance.Spec.CoreDNSCustomConfig.ConfigMapName != "" {
		err = r.removeLighthouseConfigFromCustomDNSConfigMap(ctx, instance.Spec.CoreDNSCustomConfig)
	} else {
		err = r.updateLighthouseConfigInConfigMap(ctx, instance, DefaultCoreDNSNamespace, CoreDNSName, "")
	}

	if apierrors.IsNotFound(err) {
		// Try to update Openshift-DNS
		err = r.updateLighthouseConfigInOpenshiftDNSOperator(ctx, instance, "")
	}

	if err != nil && !apierrors.IsNotFound(err) {
		return reconcile.Result{}, err
	}

	components := []*uninstall.Component{
		{
			Resource: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      names.ServiceDiscoveryComponent,
					Namespace: instance.Namespace,
				},
			},
			UninstallResource: newLighthouseAgent(instance, opnames.AppendUninstall(names.ServiceDiscoveryComponent)),
		},
	}

	uninstallInfo := &uninstall.Info{
		Client:     r.ScopedClient,
		Components: components,
		StartTime:  instance.DeletionTimestamp.Time,
		Log:        log,
		GetImageInfo: func(imageName, componentName string) (string, corev1.PullPolicy) {
			return getImagePath(instance, imageName, componentName),
				images.GetPullPolicy(instance.Spec.Version, instance.Spec.ImageOverrides[componentName])
		},
	}

	requeue, _, err := uninstallInfo.Run(ctx)
	if err != nil {
		return reconcile.Result{}, err //nolint:wrapcheck // No need to wrap
	}

	if requeue {
		return reconcile.Result{RequeueAfter: time.Millisecond * 500}, nil
	}

	return reconcile.Result{}, r.removeFinalizer(ctx, instance)
}

func (r *Reconciler) removeFinalizer(ctx context.Context, instance *operatorv1alpha1.ServiceDiscovery) error {
	return finalizer.Remove[*operatorv1alpha1.ServiceDiscovery](ctx, resource.ForControllerClient(
		r.ScopedClient, instance.Namespace, instance),
		instance, opnames.CleanupFinalizer)
}

func (r *Reconciler) removeLighthouseConfigFromCustomDNSConfigMap(ctx context.Context,
	config *operatorv1alpha1.CoreDNSCustomConfig,
) error {
	configMap := newCoreDNSCustomConfigMap(config)

	log.Info("Removing lighthouse config from custom DNS ConfigMap", "Name", configMap.Name, "Namespace", configMap.Namespace)

	err := util.Update[*corev1.ConfigMap](ctx, resource.ForControllerClient(r.GeneralClient, configMap.Namespace, configMap), configMap,
		func(existing *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			delete(existing.Data, "lighthouse.server")
			return existing, nil
		})

	return errors.Wrapf(err, "error updating custom DNS ConfigMap %q", configMap.Name)
}
