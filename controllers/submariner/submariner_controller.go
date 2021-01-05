/*
© 2019 Red Hat, Inc. and others.

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
	"fmt"
	"reflect"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	errorutil "github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	submopv1a1 "github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/helpers"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/images"
	"github.com/submariner-io/submariner-operator/pkg/names"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
)

var log = logf.Log.WithName("controller_submariner")

// NewReconciler returns a new SubmarinerReconciler
func NewReconciler(mgr manager.Manager) *SubmarinerReconciler {
	reconciler := &SubmarinerReconciler{
		client:         mgr.GetClient(),
		config:         mgr.GetConfig(),
		log:            ctrl.Log.WithName("controllers").WithName("Submariner"),
		scheme:         mgr.GetScheme(),
		clientSet:      kubernetes.NewForConfigOrDie(mgr.GetConfig()),
		dynClient:      dynamic.NewForConfigOrDie(mgr.GetConfig()),
		submClient:     submarinerclientset.NewForConfigOrDie(mgr.GetConfig()),
		clusterNetwork: nil,
	}

	return reconciler
}

// blank assignment to verify that SubmarinerReconciler implements reconcile.Reconciler
var _ reconcile.Reconciler = &SubmarinerReconciler{}

// SubmarinerReconciler reconciles a Submariner object
type SubmarinerReconciler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client         client.Client
	config         *rest.Config
	log            logr.Logger
	scheme         *runtime.Scheme
	clientSet      kubernetes.Interface
	submClient     submarinerclientset.Interface
	dynClient      dynamic.Interface
	clusterNetwork *network.ClusterNetwork
}

// Reconcile reads that state of the cluster for a Submariner object and makes changes based on the state read
// and what is in the Submariner.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.

// +kubebuilder:rbac:groups=submariner.io,resources=submariners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=submariner.io,resources=submariners/status,verbs=get;update;patch
func (r *SubmarinerReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Submariner")

	// Fetch the Submariner instance
	instance := &submopv1a1.Submariner{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
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

	initialStatus := instance.Status.DeepCopy()

	instance.SetDefaults()

	if err := r.client.Update(context.TODO(), instance); err != nil {
		return reconcile.Result{}, err
	}

	// discovery is performed after Update to avoid storing the discovery in the
	// struct beyond status
	clusterNetwork, err := r.discoverNetwork(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	engineDaemonSet, err := r.reconcileEngineDaemonSet(instance, reqLogger)
	if err != nil {
		return reconcile.Result{}, err
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

	if _, err := r.reconcileNetworkPluginSyncerDeployment(instance, clusterNetwork, reqLogger); err != nil {
		return reconcile.Result{}, err
	}

	// Retrieve the gateway information
	gateways, err := r.retrieveGateways(instance, request.Namespace)
	if err != nil {
		// Not fatal
		log.Error(err, "error retrieving gateways")
	}

	var gatewayStatuses = []submv1.GatewayStatus{}
	if gateways != nil {
		recordGateways(len(*gateways))
		// Clear the connections so we don’t remember stale status information
		recordNoConnections()
		for _, gateway := range *gateways {
			gatewayStatuses = append(gatewayStatuses, gateway.Status)
			recordGatewayCreationTime(gateway.Status.LocalEndpoint, gateway.CreationTimestamp.Time)

			for j := range gateway.Status.Connections {
				recordConnection(
					gateway.Status.LocalEndpoint,
					gateway.Status.Connections[j].Endpoint,
					string(gateway.Status.Connections[j].Status),
				)
			}
		}
	} else {
		recordGateways(0)
		recordNoConnections()
	}

	instance.Status.NatEnabled = instance.Spec.NatEnabled
	instance.Status.ColorCodes = instance.Spec.ColorCodes
	instance.Status.ClusterID = instance.Spec.ClusterID
	instance.Status.GlobalCIDR = instance.Spec.GlobalCIDR
	instance.Status.Gateways = &gatewayStatuses

	err = r.updateDaemonSetStatus(engineDaemonSet, &instance.Status.EngineDaemonSetStatus, request.Namespace)
	if err != nil {
		reqLogger.Error(err, "failed to check engine daemonset containers")
		return reconcile.Result{}, err
	}
	err = r.updateDaemonSetStatus(routeagentDaemonSet, &instance.Status.RouteAgentDaemonSetStatus, request.Namespace)
	if err != nil {
		reqLogger.Error(err, "failed to check route agent daemonset containers")
		return reconcile.Result{}, err
	}
	err = r.updateDaemonSetStatus(globalnetDaemonSet, &instance.Status.GlobalnetDaemonSetStatus, request.Namespace)
	if err != nil {
		reqLogger.Error(err, "failed to check engine daemonset containers")
		return reconcile.Result{}, err
	}
	if !reflect.DeepEqual(instance.Status, initialStatus) {
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "failed to update the Submariner status")
			return reconcile.Result{}, err
		}
	}

	if err := r.serviceDiscoveryReconciler(instance, reqLogger, instance.Spec.ServiceDiscoveryEnabled); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *SubmarinerReconciler) updateDaemonSetStatus(daemonSet *appsv1.DaemonSet, status *submopv1a1.DaemonSetStatus,
	namespace string) error {
	if daemonSet != nil {
		if status == nil {
			status = &submopv1a1.DaemonSetStatus{}
		}
		status.Status = &daemonSet.Status
		if status.LastResourceVersion != daemonSet.ObjectMeta.ResourceVersion {
			// The daemonset has changed, check its containers
			mismatchedContainerImages, nonReadyContainerStates, err :=
				r.checkDaemonSetContainers(daemonSet, namespace)
			if err != nil {
				return err
			}
			status.MismatchedContainerImages = mismatchedContainerImages
			status.NonReadyContainerStates = nonReadyContainerStates
			status.LastResourceVersion = daemonSet.ObjectMeta.ResourceVersion
		}
	}
	return nil
}

func (r *SubmarinerReconciler) checkDaemonSetContainers(daemonSet *appsv1.DaemonSet,
	namespace string) (bool, *[]corev1.ContainerState, error) {
	containerStatuses, err := r.retrieveDaemonSetContainerStatuses(daemonSet, namespace)
	if err != nil {
		return false, nil, err
	}
	var containerImageManifest *string = nil
	var mismatchedContainerImages = false
	var nonReadyContainerStates = []corev1.ContainerState{}
	for i := range *containerStatuses {
		if containerImageManifest == nil {
			containerImageManifest = &((*containerStatuses)[i].ImageID)
		} else if *containerImageManifest != (*containerStatuses)[i].ImageID {
			// Container mismatch
			mismatchedContainerImages = true
		}
		if !*(*containerStatuses)[i].Started {
			// Not (yet) ready
			nonReadyContainerStates = append(nonReadyContainerStates, (*containerStatuses)[i].State)
		}
	}
	return mismatchedContainerImages, &nonReadyContainerStates, nil
}

func (r *SubmarinerReconciler) retrieveDaemonSetContainerStatuses(daemonSet *appsv1.DaemonSet,
	namespace string) (*[]corev1.ContainerStatus, error) {
	pods := &corev1.PodList{}
	selector, err := metav1.LabelSelectorAsSelector(daemonSet.Spec.Selector)
	if err != nil {
		return nil, err
	}
	err = r.client.List(context.TODO(), pods, client.InNamespace(namespace), client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return nil, err
	}
	containerStatuses := []corev1.ContainerStatus{}
	for i := range pods.Items {
		containerStatuses = append(containerStatuses, pods.Items[i].Status.ContainerStatuses...)
	}
	return &containerStatuses, nil
}

func (r *SubmarinerReconciler) retrieveGateways(owner metav1.Object, namespace string) (*[]submv1.Gateway, error) {
	foundGateways := &submv1.GatewayList{}
	err := r.client.List(context.TODO(), foundGateways, client.InNamespace(namespace))
	if err != nil && errors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Ensure we’ll get updates
	for i := range foundGateways.Items {
		if err := controllerutil.SetControllerReference(owner, &foundGateways.Items[i], r.scheme); err != nil {
			return nil, err
		}
	}
	return &foundGateways.Items, nil
}

func (r *SubmarinerReconciler) reconcileEngineDaemonSet(instance *submopv1a1.Submariner, reqLogger logr.Logger) (*appsv1.DaemonSet, error) {
	daemonSet, err := helpers.ReconcileDaemonSet(instance, newEngineDaemonSet(instance), reqLogger, r.client, r.scheme)
	if err != nil {
		return nil, err
	}
	_, err = r.setupMetrics(instance, daemonSet.GetLabels(), reqLogger)
	return daemonSet, err
}

func (r *SubmarinerReconciler) reconcileNetworkPluginSyncerDeployment(instance *submopv1a1.Submariner,
	clusterNetwork *network.ClusterNetwork, reqLogger logr.Logger) (*appsv1.Deployment, error) {
	// Only OVNKubernetes needs networkplugin-syncer so far
	if instance.Status.NetworkPlugin == network.OvnKubernetes {
		return helpers.ReconcileDeployment(instance, newNetworkPluginSyncerDeployment(instance,
			clusterNetwork), reqLogger, r.client, r.scheme)
	}
	return nil, nil
}

func (r *SubmarinerReconciler) reconcileRouteagentDaemonSet(instance *submopv1a1.Submariner, reqLogger logr.Logger) (*appsv1.DaemonSet,
	error) {
	return helpers.ReconcileDaemonSet(instance, newRouteAgentDaemonSet(instance), reqLogger, r.client, r.scheme)
}

func (r *SubmarinerReconciler) reconcileGlobalnetDaemonSet(instance *submopv1a1.Submariner, reqLogger logr.Logger) (*appsv1.DaemonSet,
	error) {
	return helpers.ReconcileDaemonSet(instance, newGlobalnetDaemonSet(instance), reqLogger, r.client, r.scheme)
}

func (r *SubmarinerReconciler) serviceDiscoveryReconciler(submariner *submopv1a1.Submariner, reqLogger logr.Logger, isEnabled bool) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if isEnabled {
			sd := newServiceDiscoveryCR(submariner.Namespace)
			result, err := controllerutil.CreateOrUpdate(context.TODO(), r.client, sd, func() error {
				sd.Spec = submopv1a1.ServiceDiscoverySpec{
					Version:                  submariner.Spec.Version,
					Repository:               submariner.Spec.Repository,
					BrokerK8sCA:              submariner.Spec.BrokerK8sCA,
					BrokerK8sRemoteNamespace: submariner.Spec.BrokerK8sRemoteNamespace,
					BrokerK8sApiServerToken:  submariner.Spec.BrokerK8sApiServerToken,
					BrokerK8sApiServer:       submariner.Spec.BrokerK8sApiServer,
					Debug:                    submariner.Spec.Debug,
					ClusterID:                submariner.Spec.ClusterID,
					Namespace:                submariner.Spec.Namespace,
					GlobalnetEnabled:         submariner.Spec.GlobalCIDR != "",
					ImageOverrides:           submariner.Spec.ImageOverrides,
				}
				if len(submariner.Spec.CustomDomains) > 0 {
					sd.Spec.CustomDomains = submariner.Spec.CustomDomains
				}
				// Set the owner and controller
				return controllerutil.SetControllerReference(submariner, sd, r.scheme)
			})
			if err != nil {
				return err
			}
			if result == controllerutil.OperationResultCreated {
				reqLogger.Info("Created Service Discovery CR", "Namespace", sd.Namespace, "Name", sd.Name)
			} else if result == controllerutil.OperationResultUpdated {
				reqLogger.Info("Updated Service Discovery CR", "Namespace", sd.Namespace, "Name", sd.Name)
			}
			return err
		} else {
			sdExisting := newServiceDiscoveryCR(submariner.Namespace)
			err := r.client.Delete(context.TODO(), sdExisting)
			if errors.IsNotFound(err) {
				return nil
			} else if err == nil {
				reqLogger.Info("Deleted Service Discovery CR", "Namespace", submariner.Namespace)
			}
			return err
		}
	})

	return errorutil.WithMessagef(err, "error reconciling the Service Discovery CR")
}

func (r *SubmarinerReconciler) setupMetrics(instance *submopv1a1.Submariner, labels map[string]string,
	reqLogger logr.Logger) (*corev1.Service, error) {
	app, ok := labels["app"]
	if !ok {
		return nil, fmt.Errorf("No app label in the provided labels, %v", labels)
	}
	metricsService, err := helpers.ReconcileService(instance, newMetricsService(instance, app), reqLogger, r.client, r.scheme)
	if err != nil {
		return nil, err
	}

	if r.config != nil {
		services := []*corev1.Service{metricsService}
		_, err = metrics.CreateServiceMonitors(r.config, instance.Namespace, services)
		if err != nil {
			log.Info("Could not create ServiceMonitor object", "error", err.Error())
			// If this operator is deployed to a cluster without the prometheus-operator running, it will return
			// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
			if err == metrics.ErrServiceMonitorNotPresent {
				log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
			} else if !errors.IsAlreadyExists(err) {
				return nil, err
			}
		}
	}

	return metricsService, nil
}

func newEngineDaemonSet(cr *submopv1a1.Submariner) *appsv1.DaemonSet {
	labels := map[string]string{
		"app":       "submariner-engine",
		"component": "engine",
	}

	revisionHistoryLimit := int32(5)

	maxUnavailable := intstr.FromInt(1)

	deployment := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Namespace: cr.Namespace,
			Name:      "submariner-gateway",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "submariner-engine"}},
			Template: newEnginePodTemplate(cr),
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &maxUnavailable,
				},
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
			},
			RevisionHistoryLimit: &revisionHistoryLimit,
		},
	}

	return deployment
}

// newMetricsService populates a Service providing access to metrics for the given application
// The assumptions are:
// - the application's resources are labeled with "app=" the given app name
// - the metrics are exposed on port 8080 on "/metrics"
// The Service is named after the application name, suffixed with "-metrics"
func newMetricsService(cr *submopv1a1.Submariner, app string) *corev1.Service {
	labels := map[string]string{
		"app": app,
	}

	servicePorts := []corev1.ServicePort{
		{Port: 8080, Name: "metrics", Protocol: corev1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int,
			IntVal: 8080}},
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Namespace: cr.Namespace,
			Name:      fmt.Sprintf("%s-metrics", app),
		},
		Spec: corev1.ServiceSpec{
			Ports:    servicePorts,
			Selector: labels,
		},
	}

	return service
}

// newEnginePodTemplate returns a submariner pod with the same fields as the cr
func newEnginePodTemplate(cr *submopv1a1.Submariner) corev1.PodTemplateSpec {
	labels := map[string]string{
		"app": "submariner-engine",
	}

	// Create privileged security context for Engine pod
	// FIXME: Seems like these have to be a var, so can pass pointer to bool var to SecurityContext. Cleaner option?
	allowPrivilegeEscalation := true
	privileged := true
	runAsNonRoot := false
	readOnlyRootFilesystem := false

	security_context_all_caps_privileged := corev1.SecurityContext{
		Capabilities:             &corev1.Capabilities{Add: []corev1.Capability{"ALL"}},
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		Privileged:               &privileged,
		ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
		RunAsNonRoot:             &runAsNonRoot}

	// Create Pod
	terminationGracePeriodSeconds := int64(1)

	// Default healthCheck Values
	healthCheckEnabled := true
	// The values are in seconds
	healthCheckInterval := uint64(1)
	healthCheckMaxPacketLossCount := uint64(5)

	if cr.Spec.ConnectionHealthCheck != nil {
		healthCheckEnabled = cr.Spec.ConnectionHealthCheck.Enabled
		healthCheckInterval = cr.Spec.ConnectionHealthCheck.IntervalSeconds
		healthCheckMaxPacketLossCount = cr.Spec.ConnectionHealthCheck.MaxPacketLossCount
	}

	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
		Spec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: labels,
						},
						TopologyKey: "kubernetes.io/hostname",
					}},
				},
			},
			NodeSelector: map[string]string{"submariner.io/gateway": "true"},
			Containers: []corev1.Container{
				{
					Name:            "submariner",
					Image:           getImagePath(cr, names.EngineImage),
					ImagePullPolicy: helpers.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.EngineImage]),
					Command:         []string{"submariner.sh"},
					SecurityContext: &security_context_all_caps_privileged,
					VolumeMounts: []corev1.VolumeMount{
						{Name: "host-slash", MountPath: "/host", ReadOnly: true},
					},
					Env: []corev1.EnvVar{
						{Name: "SUBMARINER_NAMESPACE", Value: cr.Spec.Namespace},
						{Name: "SUBMARINER_CLUSTERCIDR", Value: cr.Status.ClusterCIDR},
						{Name: "SUBMARINER_SERVICECIDR", Value: cr.Status.ServiceCIDR},
						{Name: "SUBMARINER_GLOBALCIDR", Value: cr.Spec.GlobalCIDR},
						{Name: "SUBMARINER_CLUSTERID", Value: cr.Spec.ClusterID},
						{Name: "SUBMARINER_COLORCODES", Value: cr.Spec.ColorCodes},
						{Name: "SUBMARINER_DEBUG", Value: strconv.FormatBool(cr.Spec.Debug)},
						{Name: "SUBMARINER_NATENABLED", Value: strconv.FormatBool(cr.Spec.NatEnabled)},
						{Name: "SUBMARINER_BROKER", Value: cr.Spec.Broker},
						{Name: "SUBMARINER_CABLEDRIVER", Value: cr.Spec.CableDriver},
						{Name: "BROKER_K8S_APISERVER", Value: cr.Spec.BrokerK8sApiServer},
						{Name: "BROKER_K8S_APISERVERTOKEN", Value: cr.Spec.BrokerK8sApiServerToken},
						{Name: "BROKER_K8S_REMOTENAMESPACE", Value: cr.Spec.BrokerK8sRemoteNamespace},
						{Name: "BROKER_K8S_CA", Value: cr.Spec.BrokerK8sCA},
						{Name: "CE_IPSEC_PSK", Value: cr.Spec.CeIPSecPSK},
						{Name: "CE_IPSEC_DEBUG", Value: strconv.FormatBool(cr.Spec.CeIPSecDebug)},
						{Name: "SUBMARINER_HEALTHCHECKENABLED", Value: strconv.FormatBool(healthCheckEnabled)},
						{Name: "SUBMARINER_HEALTHCHECKINTERVAL", Value: strconv.FormatUint(healthCheckInterval, 10)},
						{Name: "SUBMARINER_HEALTHCHECKMAXPACKETLOSSCOUNT", Value: strconv.FormatUint(healthCheckMaxPacketLossCount, 10)},
						{Name: "NODE_NAME", ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "spec.nodeName",
							},
						}},
					},
				},
			},
			// TODO: Use SA submariner-engine or submariner?
			ServiceAccountName:            "submariner-operator",
			HostNetwork:                   true,
			TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
			RestartPolicy:                 corev1.RestartPolicyAlways,
			DNSPolicy:                     corev1.DNSClusterFirst,
			Volumes: []corev1.Volume{
				{Name: "host-slash", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/"}}},
			},
			// The gateway engine must be able to run on any flagged node, regardless of existing taints
			Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
		},
	}
	if cr.Spec.CeIPSecIKEPort != 0 {
		podTemplate.Spec.Containers[0].Env = append(podTemplate.Spec.Containers[0].Env,
			corev1.EnvVar{Name: "CE_IPSEC_IKEPORT", Value: strconv.Itoa(cr.Spec.CeIPSecIKEPort)})
	}

	if cr.Spec.CeIPSecNATTPort != 0 {
		podTemplate.Spec.Containers[0].Env = append(podTemplate.Spec.Containers[0].Env,
			corev1.EnvVar{Name: "CE_IPSEC_NATTPORT", Value: strconv.Itoa(cr.Spec.CeIPSecNATTPort)})
	}

	return podTemplate
}

func newRouteAgentDaemonSet(cr *submopv1a1.Submariner) *appsv1.DaemonSet {
	labels := map[string]string{
		"app":       "submariner-routeagent",
		"component": "routeagent",
	}

	matchLabels := map[string]string{
		"app": "submariner-routeagent",
	}

	allowPrivilegeEscalation := true
	privileged := true
	readOnlyFileSystem := false
	runAsNonRoot := false
	security_context_all_cap_allow_escal := corev1.SecurityContext{
		Capabilities:             &corev1.Capabilities{Add: []corev1.Capability{"ALL"}},
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		Privileged:               &privileged,
		ReadOnlyRootFilesystem:   &readOnlyFileSystem,
		RunAsNonRoot:             &runAsNonRoot,
	}

	terminationGracePeriodSeconds := int64(1)
	maxUnavailable := intstr.FromString("100%")

	routeAgentDaemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      "submariner-routeagent",
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: matchLabels},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &maxUnavailable,
				},
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Containers: []corev1.Container{
						{
							Name:            "submariner-routeagent",
							Image:           getImagePath(cr, names.RouteAgentImage),
							ImagePullPolicy: helpers.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.RouteAgentImage]),
							// FIXME: Should be entrypoint script, find/use correct file for routeagent
							Command:         []string{"submariner-route-agent.sh"},
							SecurityContext: &security_context_all_cap_allow_escal,
							VolumeMounts: []corev1.VolumeMount{
								{Name: "host-slash", MountPath: "/host", ReadOnly: true},
							},
							Env: []corev1.EnvVar{
								{Name: "SUBMARINER_NAMESPACE", Value: cr.Spec.Namespace},
								{Name: "SUBMARINER_CLUSTERID", Value: cr.Spec.ClusterID},
								{Name: "SUBMARINER_DEBUG", Value: strconv.FormatBool(cr.Spec.Debug)},
								{Name: "SUBMARINER_CLUSTERCIDR", Value: cr.Status.ClusterCIDR},
								{Name: "SUBMARINER_SERVICECIDR", Value: cr.Status.ServiceCIDR},
								{Name: "SUBMARINER_GLOBALCIDR", Value: cr.Spec.GlobalCIDR},
								{Name: "SUBMARINER_NETWORKPLUGIN", Value: cr.Status.NetworkPlugin},
								{Name: "NODE_NAME", ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "spec.nodeName",
									},
								}},
							},
						},
					},
					// TODO: Use SA submariner-routeagent or submariner?
					ServiceAccountName: "submariner-operator",
					HostNetwork:        true,
					Volumes: []corev1.Volume{
						{Name: "host-slash", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/"}}},
					},
					// The route agent engine on all nodes, regardless of existing taints
					Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
				},
			},
		},
	}

	return routeAgentDaemonSet
}

func newGlobalnetDaemonSet(cr *submopv1a1.Submariner) *appsv1.DaemonSet {
	labels := map[string]string{
		"app":       "submariner-globalnet",
		"component": "globalnet",
	}

	matchLabels := map[string]string{
		"app": "submariner-globalnet",
	}

	allowPrivilegeEscalation := true
	privileged := true
	readOnlyFileSystem := false
	runAsNonRoot := false
	security_context_all_cap_allow_escal := corev1.SecurityContext{
		Capabilities:             &corev1.Capabilities{Add: []corev1.Capability{"ALL"}},
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		Privileged:               &privileged,
		ReadOnlyRootFilesystem:   &readOnlyFileSystem,
		RunAsNonRoot:             &runAsNonRoot,
	}

	terminationGracePeriodSeconds := int64(2)

	globalnetDaemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      "submariner-globalnet",
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: matchLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "submariner-globalnet",
							Image:           getImagePath(cr, names.GlobalnetImage),
							ImagePullPolicy: helpers.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.GlobalnetImage]),
							SecurityContext: &security_context_all_cap_allow_escal,
							VolumeMounts: []corev1.VolumeMount{
								{Name: "host-slash", MountPath: "/host", ReadOnly: true},
							},
							Env: []corev1.EnvVar{
								{Name: "SUBMARINER_NAMESPACE", Value: cr.Spec.Namespace},
								{Name: "SUBMARINER_CLUSTERID", Value: cr.Spec.ClusterID},
								{Name: "SUBMARINER_EXCLUDENS", Value: "submariner-operator,kube-system,operators,openshift-monitoring,openshift-dns"},
								{Name: "NODE_NAME", ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "spec.nodeName",
									},
								}},
							},
						},
					},
					// TODO: Use SA submariner-globalnet or submariner?
					ServiceAccountName:            "submariner-operator",
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					NodeSelector:                  map[string]string{"submariner.io/gateway": "true"},
					HostNetwork:                   true,
					Volumes: []corev1.Volume{
						{Name: "host-slash", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/"}}},
					},
					// The Globalnet Pod must be able to run on any flagged node, regardless of existing taints
					Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
				},
			},
		},
	}

	return globalnetDaemonSet
}

func newServiceDiscoveryCR(namespace string) *submopv1a1.ServiceDiscovery {
	return &submopv1a1.ServiceDiscovery{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      names.ServiceDiscoveryCrName,
		},
	}
}

func getImagePath(submariner *submopv1a1.Submariner, componentImage string) string {
	return images.GetImagePath(submariner.Spec.Repository, submariner.Spec.Version, componentImage,
		submariner.Spec.ImageOverrides)
}

func (r *SubmarinerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// These are required so that we can retrieve Gateway objects using the dynamic client
	if err := submv1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	// These are required so that we can manipulate CRDs
	if err := apiextensions.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	// Create a new controller
	c, err := controller.New("submariner-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Submariner
	err = c.Watch(&source.Kind{Type: &submopv1a1.Submariner{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource DaemonSets and requeue the owner Submariner
	err = c.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &submopv1a1.Submariner{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to the gateway status in the same namespace
	mapFn := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Name:      "submariner",
					Namespace: a.Meta.GetNamespace(),
				}},
			}
		})
	err = c.Watch(&source.Kind{Type: &submv1.Gateway{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: mapFn,
	})
	if err != nil {
		log.Error(err, "error watching gateways")
		// This isn’t fatal
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&submopv1a1.Submariner{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
