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

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/names"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) serviceDiscoveryReconciler(ctx context.Context, submariner *v1alpha1.Submariner, reqLogger logr.Logger,
	isEnabled bool,
) error {
	// nolint:wrapcheck // No need to wrap errors here
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if isEnabled {
			sd := newServiceDiscoveryCR(submariner.Namespace)
			result, err := controllerutil.CreateOrUpdate(ctx, r.config.ScopedClient, sd, func() error {
				sd.Spec = v1alpha1.ServiceDiscoverySpec{
					Version:                  submariner.Spec.Version,
					Repository:               submariner.Spec.Repository,
					BrokerK8sCA:              submariner.Spec.BrokerK8sCA,
					BrokerK8sRemoteNamespace: submariner.Spec.BrokerK8sRemoteNamespace,
					BrokerK8sApiServerToken:  submariner.Spec.BrokerK8sApiServerToken,
					BrokerK8sApiServer:       submariner.Spec.BrokerK8sApiServer,
					BrokerK8sInsecure:        submariner.Spec.BrokerK8sInsecure,
					Debug:                    submariner.Spec.Debug,
					ClusterID:                submariner.Spec.ClusterID,
					Namespace:                submariner.Spec.Namespace,
					GlobalnetEnabled:         submariner.Spec.GlobalCIDR != "",
					ImageOverrides:           submariner.Spec.ImageOverrides,
					CoreDNSCustomConfig:      submariner.Spec.CoreDNSCustomConfig,
				}

				if len(submariner.Spec.CustomDomains) > 0 {
					sd.Spec.CustomDomains = submariner.Spec.CustomDomains
				}
				// Set the owner and controller
				return controllerutil.SetControllerReference(submariner, sd, r.config.Scheme)
			})
			if err != nil {
				return err
			}

			if result == controllerutil.OperationResultCreated {
				reqLogger.Info("Created Service Discovery CR", "Namespace", sd.Namespace, "Name", sd.Name)
			} else if result == controllerutil.OperationResultUpdated {
				reqLogger.Info("Updated Service Discovery CR", "Namespace", sd.Namespace, "Name", sd.Name)
			}

			return nil
		}

		sdExisting := newServiceDiscoveryCR(submariner.Namespace)
		err := r.config.ScopedClient.Delete(ctx, sdExisting)
		if apierrors.IsNotFound(err) {
			return nil
		} else if err == nil {
			reqLogger.Info("Deleted Service Discovery CR", "Namespace", submariner.Namespace)
		}

		return err
	})

	return errors.Wrapf(err, "error reconciling the Service Discovery CR")
}

func newServiceDiscoveryCR(namespace string) *v1alpha1.ServiceDiscovery {
	return &v1alpha1.ServiceDiscovery{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      names.ServiceDiscoveryCrName,
		},
	}
}
