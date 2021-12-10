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
	"reflect"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	submopv1a1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/gateway"
	"github.com/submariner-io/submariner-operator/pkg/images"
	crdutils "github.com/submariner-io/submariner-operator/pkg/utils/crds"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	gatewayMetricsServerPort   = 8080
	globalnetMetricsServerPort = 8081
)

var log = logf.Log.WithName("controller_submariner")

type Config struct {
	// This client is a split client that reads objects from the cache and writes to the apiserver
	Client         client.Client
	RestConfig     *rest.Config
	Scheme         *runtime.Scheme
	KubeClient     kubernetes.Interface
	SubmClient     submarinerclientset.Interface
	DynClient      dynamic.Interface
	ClusterNetwork *network.ClusterNetwork
}

// Reconciler reconciles a Submariner object.
type Reconciler struct {
	config Config
	log    logr.Logger
}

// blank assignment to verify that Reconciler implements reconcile.Reconciler.
var _ reconcile.Reconciler = &Reconciler{}

// NewReconciler returns a new Reconciler.
func NewReconciler(config *Config) *Reconciler {
	return &Reconciler{
		config: *config,
		log:    ctrl.Log.WithName("controllers").WithName("Submariner"),
	}
}

// Reconcile reads that state of the cluster for a Submariner object and makes changes based on the state read
// and what is in the Submariner.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.

// +kubebuilder:rbac:groups=submariner.io,resources=submariners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=submariner.io,resources=submariners/status,verbs=get;update;patch
// nolint:gocyclo // Refactoring would yield functions with a lot of params which isn't ideal either.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Submariner")

	// Fetch the Submariner instance
	instance := &submopv1a1.Submariner{}
	err := r.config.Client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, errors.Wrap(err, "error retrieving Submariner resource")
	}

	if instance.ObjectMeta.DeletionTimestamp != nil {
		// Graceful deletion has been requested, ignore the object
		return reconcile.Result{}, nil
	}

	initialStatus := instance.Status.DeepCopy()

	clusterNetwork, err := r.discoverNetwork(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	gatewayDaemonSet, err := r.reconcileGatewayDaemonSet(instance, reqLogger)
	if err != nil {
		return reconcile.Result{}, err
	}

	var loadBalancer *corev1.Service
	if instance.Spec.LoadBalancerEnabled {
		loadBalancer, err = r.reconcileLoadBalancer(instance, reqLogger)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	routeagentDaemonSet, err := r.reconcileRouteagentDaemonSet(instance, reqLogger)
	if err != nil {
		return reconcile.Result{}, err
	}

	var globalnetDaemonSet *appsv1.DaemonSet
	if instance.Spec.GlobalCIDR != "" {
		if globalnetDaemonSet, err = r.reconcileGlobalnetDaemonSet(instance, reqLogger); err != nil {
			return reconcile.Result{}, err
		}
	}

	if err := r.reconcileNetworkPluginSyncerDeployment(instance, clusterNetwork, reqLogger); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.serviceDiscoveryReconciler(ctx, instance, reqLogger, instance.Spec.ServiceDiscoveryEnabled); err != nil {
		return reconcile.Result{}, err
	}

	// Retrieve the gateway information
	gateways, err := r.retrieveGateways(ctx, instance, request.Namespace)
	if err != nil {
		// Not fatal
		log.Error(err, "error retrieving gateways")
	}

	gatewayStatuses := buildGatewayStatusAndUpdateMetrics(gateways)

	instance.Status.NatEnabled = instance.Spec.NatEnabled
	instance.Status.ColorCodes = instance.Spec.ColorCodes
	instance.Status.ClusterID = instance.Spec.ClusterID
	instance.Status.GlobalCIDR = instance.Spec.GlobalCIDR
	instance.Status.Gateways = &gatewayStatuses

	err = updateDaemonSetStatus(ctx, r.config.Client, gatewayDaemonSet, &instance.Status.GatewayDaemonSetStatus, request.Namespace)
	if err != nil {
		reqLogger.Error(err, "failed to check gateway daemonset containers")
		return reconcile.Result{}, err
	}
	err = updateDaemonSetStatus(ctx, r.config.Client, routeagentDaemonSet, &instance.Status.RouteAgentDaemonSetStatus, request.Namespace)
	if err != nil {
		reqLogger.Error(err, "failed to check route agent daemonset containers")
		return reconcile.Result{}, err
	}
	err = updateDaemonSetStatus(ctx, r.config.Client, globalnetDaemonSet, &instance.Status.GlobalnetDaemonSetStatus, request.Namespace)
	if err != nil {
		reqLogger.Error(err, "failed to check gateway daemonset containers")
		return reconcile.Result{}, err
	}

	if loadBalancer != nil {
		instance.Status.LoadBalancerStatus.Status = &loadBalancer.Status.LoadBalancer
	} else {
		instance.Status.LoadBalancerStatus.Status = nil
	}

	if !reflect.DeepEqual(instance.Status, initialStatus) {
		err := r.config.Client.Status().Update(ctx, instance)
		if err != nil {
			reqLogger.Error(err, "failed to update the Submariner status")
			// Log the error, but indicate success, to avoid reconciliation storms
			// TODO skitt determine what we should really be doing for concurrent updates to the Submariner CR
			// Updates fail here because the instance is updated between the .Update() at the start of the function
			// and the status update here
		}
	}

	return reconcile.Result{}, nil
}

func getImagePath(submariner *submopv1a1.Submariner, imageName, componentName string) string {
	return images.GetImagePath(submariner.Spec.Repository, submariner.Spec.Version, imageName, componentName,
		submariner.Spec.ImageOverrides)
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Set up the CRDs we need
	crdUpdater, err := crdutils.NewFromRestConfig(mgr.GetConfig())
	if err != nil {
		return errors.Wrap(err, "error creating CRDUpdater")
	}
	if err := gateway.Ensure(crdUpdater); err != nil {
		return err // nolint:wrapcheck // Errors are already wrapped
	}

	// These are required so that we can retrieve Gateway objects using the dynamic client
	if err := submv1.AddToScheme(mgr.GetScheme()); err != nil {
		return errors.Wrap(err, "error adding to the scheme")
	}
	// These are required so that we can manipulate CRDs
	if err := apiextensions.AddToScheme(mgr.GetScheme()); err != nil {
		return errors.Wrap(err, "error adding to the scheme")
	}

	// Watch for changes to the gateway status in the same namespace
	mapFn := handler.MapFunc(
		func(object client.Object) []reconcile.Request {
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Name:      "submariner",
					Namespace: object.GetNamespace(),
				}},
			}
		})

	// nolint:wrapcheck // No need to wrap here
	return ctrl.NewControllerManagedBy(mgr).
		Named("submariner-controller").
		// Watch for changes to primary resource Submariner
		For(&submopv1a1.Submariner{}).
		// Watch for changes to secondary resource DaemonSets and requeue the owner Submariner
		Owns(&appsv1.DaemonSet{}).
		Watches(&source.Kind{Type: &submv1.Gateway{}}, handler.EnqueueRequestsFromMapFunc(mapFn)).
		Complete(r)
}
