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
	"sync"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/federate"
	"github.com/submariner-io/admiral/pkg/finalizer"
	level "github.com/submariner-io/admiral/pkg/log"
	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/admiral/pkg/syncer"
	"github.com/submariner-io/admiral/pkg/util"
	submopv1a1 "github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/constants"
	resourceiface "github.com/submariner-io/submariner-operator/controllers/resource"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/images"
	"github.com/submariner-io/submariner-operator/pkg/names"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	gatewayMetricsServicePort   = 8080
	globalnetMetricsServicePort = 8081
	gatewayMetricsServerPort    = "32780"
	globalnetMetricsServerPort  = "32781"
)

var log = logf.Log.WithName("controller_submariner")

type Config struct {
	// This client is scoped to the operator namespace intended to only be used for resources created and maintained by this
	// controller. Also it's a split client that reads objects from the cache and writes to the apiserver.
	ScopedClient client.Client
	// This client can be used to access any other resource not in the operator namespace.
	GeneralClient  client.Client
	RestConfig     *rest.Config
	Scheme         *runtime.Scheme
	DynClient      dynamic.Interface
	ClusterNetwork *network.ClusterNetwork
}

// Reconciler reconciles a Submariner object.
type Reconciler struct {
	config Config
	log    logr.Logger

	// We need to synchronize changes to the SA used to connect to the broker (see names.ForClusterSA), of two kinds:
	// - changes to the token in the secret used by the SA;
	// - changes to the SA itself.
	// The secrets are communicated to the pods which need them by mounting them. This ensures that changes to the secret
	// itself get propagated to any running pods, and picked up by client-go. It implies however that the secret itself
	// can't be swapped out at runtime; so pods mount a constant secret, whose name is given to them in an environment
	// variable (see Admiral), and is specified in the Submariner CR.
	// So we need to:
	// - watch for changes to the SA, and if the secret name changes (as happens if it's deleted), update the target secret
	//   using the information from the new secret;
	// - watch for changes to the secret, and if it changes, update the target secret.
	// Tokens map back to their SA, so we can do both the above by watching tokens only.
	// Since the synchronisation ends up being specific to a Submariner CR secret, we track one syncer per Submariner CR secret name.
	// We don't keep track of the secret syncers themselves, just their cancel functions.
	secretSyncCancelFuncs map[string]context.CancelFunc
	syncerMutex           sync.Mutex
}

// blank assignment to verify that Reconciler implements reconcile.Reconciler.
var _ reconcile.Reconciler = &Reconciler{}

// NewReconciler returns a new Reconciler.
func NewReconciler(config *Config) *Reconciler {
	return &Reconciler{
		config:                *config,
		log:                   ctrl.Log.WithName("controllers").WithName("Submariner"),
		secretSyncCancelFuncs: make(map[string]context.CancelFunc),
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
	reqLogger := log.V(2).WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Submariner")

	// Fetch the Submariner instance
	instance, err := r.getSubmariner(ctx, request.NamespacedName)
	if apierrors.IsNotFound(err) {
		// Request object not found, could have been deleted after reconcile request.
		// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
		return reconcile.Result{}, nil
	}

	if err != nil {
		return reconcile.Result{}, err
	}

	instance, err = r.addFinalizer(ctx, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.GetDeletionTimestamp().IsZero() {
		log.Info("Submariner is being deleted")
		r.cancelSecretSyncer(instance)

		return r.runComponentCleanup(ctx, instance)
	}

	// Ensure we have a secret syncer
	if err := r.setupSecretSyncer(instance, reqLogger, request.Namespace); err != nil {
		return reconcile.Result{}, err
	}

	initialStatus := instance.Status.DeepCopy()

	clusterNetwork, err := r.discoverNetwork(instance, reqLogger)
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

	if _, err = r.reconcileMetricsProxyDaemonSet(instance, reqLogger); err != nil {
		return reconcile.Result{}, err
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

	err = updateDaemonSetStatus(ctx, r.config.ScopedClient, gatewayDaemonSet, &instance.Status.GatewayDaemonSetStatus, request.Namespace)
	if err != nil {
		reqLogger.Error(err, "failed to check gateway daemonset containers")

		return reconcile.Result{}, err
	}

	err = updateDaemonSetStatus(ctx, r.config.ScopedClient, routeagentDaemonSet, &instance.Status.RouteAgentDaemonSetStatus, request.Namespace)
	if err != nil {
		reqLogger.Error(err, "failed to check route agent daemonset containers")

		return reconcile.Result{}, err
	}

	err = updateDaemonSetStatus(ctx, r.config.ScopedClient, globalnetDaemonSet, &instance.Status.GlobalnetDaemonSetStatus, request.Namespace)
	if err != nil {
		reqLogger.Error(err, "failed to check gateway daemonset containers")

		return reconcile.Result{}, err
	}

	// TODO: vthapar Add metrics-proxy status to Submariner CR so we can update it with daemonset status

	if loadBalancer != nil {
		instance.Status.LoadBalancerStatus.Status = &loadBalancer.Status.LoadBalancer
	} else {
		instance.Status.LoadBalancerStatus.Status = nil
	}

	if !reflect.DeepEqual(instance.Status, initialStatus) {
		err := r.config.ScopedClient.Status().Update(ctx, instance)
		if err != nil {
			// Log the error, but indicate success, to avoid reconciliation storms
			// TODO skitt determine what we should really be doing for concurrent updates to the Submariner CR
			// Updates fail here because the instance is updated between the .Update() at the start of the function
			// and the status update here
			reqLogger.Error(err, "failed to update the Submariner status")
		}
	}

	return reconcile.Result{}, nil
}

func getImagePath(submariner *submopv1a1.Submariner, imageName, componentName string) string {
	return images.GetImagePath(submariner.Spec.Repository, submariner.Spec.Version, imageName, componentName,
		submariner.Spec.ImageOverrides)
}

func (r *Reconciler) getSubmariner(ctx context.Context, key types.NamespacedName) (*submopv1a1.Submariner, error) {
	instance := &submopv1a1.Submariner{}

	err := r.config.ScopedClient.Get(ctx, key, instance)
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving Submariner resource")
	}

	return instance, nil
}

func (r *Reconciler) addFinalizer(ctx context.Context, instance *submopv1a1.Submariner) (*submopv1a1.Submariner, error) {
	added, err := finalizer.Add(ctx, resourceiface.ForControllerClient(r.config.ScopedClient, instance.Namespace, &submopv1a1.Submariner{}),
		instance, constants.CleanupFinalizer)
	if err != nil {
		return nil, err // nolint:wrapcheck // No need to wrap
	}

	if !added {
		return instance, nil
	}

	log.Info("Added finalizer")

	return r.getSubmariner(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name})
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
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

func (r *Reconciler) setupSecretSyncer(instance *submopv1a1.Submariner, logger logr.Logger, namespace string) error {
	r.syncerMutex.Lock()
	defer r.syncerMutex.Unlock()

	if instance.Spec.BrokerK8sSecret != "" {
		if _, ok := r.secretSyncCancelFuncs[instance.Spec.BrokerK8sSecret]; !ok {
			_, gvr, err := util.ToUnstructuredResource(&corev1.Secret{}, r.config.ScopedClient.RESTMapper())
			if err != nil {
				return errors.Wrap(err, "error calculating the GVR for the Secret type")
			}
			// We can't use files here, we don't have a mounted secret
			brokerConfig, _, err := resource.GetAuthorizedRestConfigFromData(
				instance.Spec.BrokerK8sApiServer,
				instance.Spec.BrokerK8sApiServerToken, // TODO Read the secret
				instance.Spec.BrokerK8sCA,
				&rest.TLSClientConfig{Insecure: instance.Spec.BrokerK8sInsecure},
				*gvr,
				instance.Spec.BrokerK8sRemoteNamespace)
			if err != nil {
				return errors.Wrap(err, "error building an authorized RestConfig for the broker")
			}

			brokerClient, err := dynamic.NewForConfig(brokerConfig)
			if err != nil {
				return errors.Wrap(err, "error building a dynamic client for the broker")
			}

			secretSyncer, err := syncer.NewResourceSyncer(
				&syncer.ResourceSyncerConfig{
					Name:            "Broker secret syncer",
					ResourceType:    &corev1.Secret{},
					SourceClient:    brokerClient,
					SourceNamespace: instance.Spec.BrokerK8sRemoteNamespace,
					Direction:       syncer.None,
					RestMapper:      r.config.ScopedClient.RESTMapper(),
					Scheme:          r.config.Scheme,
					Federator: federate.NewCreateOrUpdateFederator(
						r.config.DynClient, r.config.ScopedClient.RESTMapper(), namespace, ""),
					Transform: func(from runtime.Object, numRequeues int, op syncer.Operation) (runtime.Object, bool) {
						secret := from.(*corev1.Secret)
						logger.V(level.TRACE).Info("Transforming secret", "secret", secret)
						if saName, ok := secret.ObjectMeta.Annotations["kubernetes.io/service-account.name"]; ok &&
							saName == names.ForClusterSA(instance.Spec.ClusterID) {
							transformedSecret := &corev1.Secret{
								ObjectMeta: metav1.ObjectMeta{
									Name: instance.Spec.BrokerK8sSecret,
								},
								Type: corev1.SecretTypeOpaque,
								Data: secret.Data,
							}
							logger.V(level.TRACE).Info("Transformed secret", "transformedSecret", transformedSecret)
							return transformedSecret, false
						}
						return nil, false
					},
				})
			if err != nil {
				return errors.Wrap(err, "error building a resource syncer for secrets")
			}

			ctx, cancelFunc := context.WithCancel(context.TODO())
			if err := secretSyncer.Start(ctx.Done()); err != nil {
				cancelFunc()
				return errors.Wrap(err, "error starting the secret syncer")
			}

			r.secretSyncCancelFuncs[instance.Spec.BrokerK8sSecret] = cancelFunc
		}
	}

	return nil
}

func (r *Reconciler) cancelSecretSyncer(instance *submopv1a1.Submariner) {
	r.syncerMutex.Lock()
	defer r.syncerMutex.Unlock()

	if instance.Spec.BrokerK8sSecret != "" {
		if cancelFunc, ok := r.secretSyncCancelFuncs[instance.Spec.BrokerK8sSecret]; ok {
			cancelFunc()
			delete(r.secretSyncCancelFuncs, instance.Spec.BrokerK8sSecret)
		}
	}
}
