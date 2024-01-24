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

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/crd"
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"
	"github.com/submariner-io/submariner-operator/pkg/gateway"
	"github.com/submariner-io/submariner-operator/pkg/lighthouse"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// BrokerReconciler reconciles a Broker object.
type BrokerReconciler struct {
	Client client.Client
	Config *rest.Config
}

//+kubebuilder:rbac:groups=submariner.io,resources=brokers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=submariner.io,resources=brokers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=submariner.io,resources=brokers/finalizers,verbs=update

func (r *BrokerReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()

	// Fetch the Broker instance
	instance := &v1alpha1.Broker{}

	err := r.Client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, errors.Wrap(err, "error retrieving Broker resource")
	}

	if instance.ObjectMeta.DeletionTimestamp != nil {
		// Graceful deletion has been requested, ignore the object
		return reconcile.Result{}, nil
	}

	// Broker CRDs
	crdUpdater := crd.UpdaterFromControllerClient(r.Client)

	err = gateway.Ensure(ctx, crdUpdater)
	if err != nil {
		return ctrl.Result{}, err //nolint:wrapcheck // Errors are already wrapped
	}

	// Lighthouse CRDs
	_, err = lighthouse.Ensure(ctx, crdUpdater, lighthouse.BrokerCluster)
	if err != nil {
		return ctrl.Result{}, err //nolint:wrapcheck // Errors are already wrapped
	}

	// Globalnet
	err = globalnet.ValidateExistingGlobalNetworks(ctx, r.Client, request.Namespace)
	if err != nil {
		return ctrl.Result{}, err //nolint:wrapcheck // Errors are already wrapped
	}

	err = globalnet.CreateConfigMap(ctx, r.Client, instance.Spec.GlobalnetEnabled, instance.Spec.GlobalnetCIDRRange,
		instance.Spec.DefaultGlobalnetClusterSize, request.Namespace)
	if err != nil {
		return ctrl.Result{}, err //nolint:wrapcheck // Errors are already wrapped
	}

	return ctrl.Result{}, nil
}

//nolint:wrapcheck // No need to wrap here.
func (r *BrokerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Broker{}).
		Complete(r)
}
