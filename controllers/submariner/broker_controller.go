/*
© 2021 Red Hat, Inc. and others.

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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/engine"
	"github.com/submariner-io/submariner-operator/pkg/lighthouse"
	crdutils "github.com/submariner-io/submariner-operator/pkg/utils/crds"
)

// BrokerReconciler reconciles a Broker object
type BrokerReconciler struct {
	Client client.Client
	Config *rest.Config
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// TODO skitt: these rbac declarations (and others, see submariner_controller.go) need to be separated
// from methods in order to be taken into account; but they produce ClusterRoles, not the Roles we want
// +kubebuilder:rbac:groups=submariner.io,resources=brokers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=submariner.io,resources=brokers/status,verbs=get;update;patch
func (r *BrokerReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("broker", request.NamespacedName)

	// Fetch the Broker instance
	instance := &v1alpha1.Broker{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Broker CRDs
	crdUpdater := crdutils.NewFromControllerClient(r.Client)
	err = engine.Ensure(crdUpdater)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Lighthouse CRDs
	_, err = lighthouse.Ensure(crdUpdater, lighthouse.BrokerCluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Globalnet
	err = broker.CreateGlobalnetConfigMap(r.Config, instance.Spec.GlobalnetEnabled, instance.Spec.GlobalnetCIDRRange,
		instance.Spec.DefaultGlobalnetClusterSize, broker.SubmarinerBrokerNamespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BrokerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Broker{}).
		Complete(r)
}
