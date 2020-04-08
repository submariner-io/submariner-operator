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
	errorutil "github.com/pkg/errors"
	submarinerv1alpha1 "github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1"
	submarinerv1alpha1clientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/versions"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_submariner")

// Add creates a new Submariner Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	client, _ := submarinerv1alpha1clientset.NewForConfig(mgr.GetConfig())
	return &ReconcileSubmariner{submClientSet : client,
		client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("submariner-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Submariner
	err = c.Watch(&source.Kind{Type: &submarinerv1alpha1.Submariner{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource DaemonSets and requeue the owner Submariner
	err = c.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &submarinerv1alpha1.Submariner{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileSubmariner implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSubmariner{}

// ReconcileSubmariner reconciles a Submariner object
type ReconcileSubmariner struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	submClientSet submarinerv1alpha1clientset.Interface
	client        client.Client
	scheme        *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Submariner object and makes changes based on the state read
// and what is in the Submariner.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileSubmariner) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Submariner")

	// Fetch the Submariner instance
	instance := &submarinerv1alpha1.Submariner{}
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

	setSubmarinerDefaults(instance)
	if err = r.client.Update(context.TODO(), instance); err != nil {
		return reconcile.Result{}, err
	}

	// Create submariner-engine SA
	//subm_engine_sa := corev1.ServiceAccount{}
	//subm_engine_sa.Name = "submariner-engine"
	//reqLogger.Info("Created a new SA", "SA.Name", subm_engine_sa.Name)

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

	// Update the status
	status := submarinerv1alpha1.SubmarinerStatus{
		NatEnabled:  instance.Spec.NatEnabled,
		ColorCodes:  instance.Spec.ColorCodes,
		ClusterID:   instance.Spec.ClusterID,
		ServiceCIDR: instance.Spec.ServiceCIDR,
		ClusterCIDR: instance.Spec.ClusterCIDR,
		GlobalCIDR:  instance.Spec.GlobalCIDR,
		CableDriver: instance.Spec.CableDriver, // TODO retrieve this from the engine
	}
	if engineDaemonSet != nil {
		status.EngineDaemonSetStatus = &engineDaemonSet.Status
	}
	if routeagentDaemonSet != nil {
		status.RouteAgentDaemonSetStatus = &routeagentDaemonSet.Status
	}
	if globalnetDaemonSet != nil {
		status.GlobalnetDaemonSetStatus = &globalnetDaemonSet.Status
	}
	if !reflect.DeepEqual(instance.Status, status) {
		instance.Status = status
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "failed to update the Submariner status")
        }

	if instance.Spec.ServiceDiscoveryEnabled {
		if err = r.reconcileServiceDiscoverCR(instance, reqLogger); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileSubmariner) deletePreExistingEngineDeployment(namespace string, reqLogger logr.Logger) error {
	foundDeployment := &appsv1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: "submariner", Namespace: namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	reqLogger.Info("Deleting existing engine Deployment")
	return r.client.Delete(context.TODO(), foundDeployment)
}

func (r *ReconcileSubmariner) reconcileEngineDaemonSet(instance *submarinerv1alpha1.Submariner, reqLogger logr.Logger) (*appsv1.DaemonSet, error) {
	daemonSet, err := r.reconcileDaemonSet(instance, newEngineDaemonSet(instance), reqLogger)
	if err == nil {
		err = r.deletePreExistingEngineDeployment(instance.Namespace, reqLogger)
	}

	return daemonSet, err
}

func (r *ReconcileSubmariner) reconcileRouteagentDaemonSet(instance *submarinerv1alpha1.Submariner, reqLogger logr.Logger) (*appsv1.DaemonSet, error) {
	return r.reconcileDaemonSet(instance, newRouteAgentDaemonSet(instance), reqLogger)
}

func (r *ReconcileSubmariner) reconcileGlobalnetDaemonSet(instance *submarinerv1alpha1.Submariner, reqLogger logr.Logger) (*appsv1.DaemonSet, error) {
	return r.reconcileDaemonSet(instance, newGlobalnetDaemonSet(instance), reqLogger)
}

func (r *ReconcileSubmariner) reconcileDaemonSet(owner metav1.Object, daemonSet *appsv1.DaemonSet, reqLogger logr.Logger) (*appsv1.DaemonSet, error) {
	var err error

	// Set the owner and controller
	if err = controllerutil.SetControllerReference(owner, daemonSet, r.scheme); err != nil {
		return nil, err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		toUpdate := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{
			Name:      daemonSet.Name,
			Namespace: daemonSet.Namespace,
			Labels:    map[string]string{},
		}}

		result, err := controllerutil.CreateOrUpdate(context.TODO(), r.client, toUpdate, func() error {
			toUpdate.Spec = daemonSet.Spec
			for k, v := range daemonSet.Labels {
				toUpdate.Labels[k] = v
			}
			return nil
		})

		if err != nil {
			return err
		}

		if result == controllerutil.OperationResultCreated {
			reqLogger.Info("Created a new DaemonSet", "DaemonSet.Namespace", daemonSet.Namespace, "DaemonSet.Name", daemonSet.Name)
		} else if result == controllerutil.OperationResultUpdated {
			reqLogger.Info("Updated existing DaemonSet", "DaemonSet.Namespace", daemonSet.Namespace, "DaemonSet.Name", daemonSet.Name)
		}

		return nil
	})

	// Update the status from the server
	if err == nil {
		err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: daemonSet.Namespace, Name: daemonSet.Name}, daemonSet)
	}

	return daemonSet, errorutil.WithMessagef(err, "error creating or updating DaemonSet %s/%s", daemonSet.Namespace, daemonSet.Name)
}

func (r *ReconcileSubmariner) reconcileServiceDiscoverCR(submariner *submarinerv1alpha1.Submariner, reqLogger logr.Logger) error {
	sd := newServiceDiscoveryCR(submariner)
	_, err := r.submClientSet.SubmarinerV1alpha1().ServiceDiscoveries(submariner.Namespace).Create(sd)
	if err != nil {
		reqLogger.Error(err, "Error creating service discovery CR")
	}
	return err
}

func newEngineDaemonSet(cr *submarinerv1alpha1.Submariner) *appsv1.DaemonSet {
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

// newEnginePodTemplate returns a submariner pod with the same fields as the cr
func newEnginePodTemplate(cr *submarinerv1alpha1.Submariner) corev1.PodTemplateSpec {
	labels := map[string]string{
		"app": "submariner-engine",
	}

	// Create privilaged security context for Engine pod
	// FIXME: Seems like these have to be a var, so can pass pointer to bool var to SecurityContext. Cleaner option?
	allowPrivilegeEscalation := true
	privileged := true
	runAsNonRoot := false
	readOnlyRootFilesystem := false

	security_context_all_caps_privilaged := corev1.SecurityContext{
		Capabilities:             &corev1.Capabilities{Add: []corev1.Capability{"ALL"}},
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		Privileged:               &privileged,
		ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
		RunAsNonRoot:             &runAsNonRoot}

	// Create Pod
	terminationGracePeriodSeconds := int64(0)
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
					Image:           getImagePath(cr, engineImage),
					Command:         []string{"submariner.sh"},
					SecurityContext: &security_context_all_caps_privilaged,
					Env: []corev1.EnvVar{
						{Name: "SUBMARINER_NAMESPACE", Value: cr.Spec.Namespace},
						{Name: "SUBMARINER_CLUSTERCIDR", Value: cr.Spec.ClusterCIDR},
						{Name: "SUBMARINER_SERVICECIDR", Value: cr.Spec.ServiceCIDR},
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
					},
				},
			},
			// TODO: Use SA submariner-engine or submariner?
			ServiceAccountName:            "submariner-operator",
			HostNetwork:                   true,
			TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
			RestartPolicy:                 corev1.RestartPolicyAlways,
			DNSPolicy:                     corev1.DNSClusterFirst,
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

func newRouteAgentDaemonSet(cr *submarinerv1alpha1.Submariner) *appsv1.DaemonSet {
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

	terminationGracePeriodSeconds := int64(0)

	routeAgentDaemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      "submariner-routeagent",
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: matchLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Containers: []corev1.Container{
						{
							Name:  "submariner-routeagent",
							Image: getImagePath(cr, routeAgentImage),
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
								{Name: "SUBMARINER_CLUSTERCIDR", Value: cr.Spec.ClusterCIDR},
								{Name: "SUBMARINER_SERVICECIDR", Value: cr.Spec.ServiceCIDR},
								{Name: "SUBMARINER_GLOBALCIDR", Value: cr.Spec.GlobalCIDR},
							},
						},
					},
					// TODO: Use SA submariner-routeagent or submariner?
					ServiceAccountName: "submariner-operator",
					HostNetwork:        true,
					Volumes: []corev1.Volume{
						{Name: "host-slash", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/"}}},
					},
				},
			},
		},
	}

	return routeAgentDaemonSet
}

func newGlobalnetDaemonSet(cr *submarinerv1alpha1.Submariner) *appsv1.DaemonSet {
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

	terminationGracePeriodSeconds := int64(0)

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
							Image:           getImagePath(cr, globalnetImage),
							ImagePullPolicy: "IfNotPresent",
							SecurityContext: &security_context_all_cap_allow_escal,
							VolumeMounts: []corev1.VolumeMount{
								{Name: "host-slash", MountPath: "/host", ReadOnly: true},
							},
							Env: []corev1.EnvVar{
								{Name: "SUBMARINER_NAMESPACE", Value: cr.Spec.Namespace},
								{Name: "SUBMARINER_CLUSTERID", Value: cr.Spec.ClusterID},
								{Name: "SUBMARINER_EXCLUDENS", Value: "submariner,kube-system,operators"},
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
				},
			},
		},
	}

	return globalnetDaemonSet
}

func newServiceDiscoveryCR(cr *submarinerv1alpha1.Submariner) *submarinerv1alpha1.ServiceDiscovery {

	deployment := &submarinerv1alpha1.ServiceDiscovery{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      "service-discovery",
		},
		Spec: submarinerv1alpha1.ServiceDiscoverySpec{
			Version:                  cr.Spec.Version,
			Repository:               cr.Spec.Repository,
			BrokerK8sCA:              cr.Spec.BrokerK8sCA,
			BrokerK8sRemoteNamespace: cr.Spec.BrokerK8sRemoteNamespace,
			BrokerK8sApiServerToken:  cr.Spec.BrokerK8sApiServerToken,
			BrokerK8sApiServer:       cr.Spec.BrokerK8sApiServer,
			Debug:                    cr.Spec.Debug,
			Broker:                   cr.Spec.Broker,
			ClusterID:                cr.Spec.ClusterID,
			Namespace:                cr.Spec.Namespace,
		},
	}

	return deployment
}

//TODO: move to a method on the API definitions, as the example shown by the etcd operator here :
//      https://github.com/coreos/etcd-operator/blob/8347d27afa18b6c76d4a8bb85ad56a2e60927018/pkg/apis/etcd/v1beta2/cluster.go#L185
func setSubmarinerDefaults(submariner *submarinerv1alpha1.Submariner) {

	if submariner.Spec.Repository == "" {
		// An empty field is converted to the default upstream submariner repository where all images live
		submariner.Spec.Repository = versions.DefaultSubmarinerRepo
	}

	if submariner.Spec.Version == "" {
		submariner.Spec.Version = versions.DefaultSubmarinerVersion
	}

	if submariner.Spec.ColorCodes == "" {
		submariner.Spec.ColorCodes = "blue"
	}

}

const (
	routeAgentImage = "submariner-route-agent"
	engineImage     = "submariner"
	globalnetImage  = "submariner-globalnet"
)

func getImagePath(submariner *submarinerv1alpha1.Submariner, componentImage string) string {
	var path string
	spec := submariner.Spec

	// If the repository is "local" we don't append it on the front of the image,
	// a local repository is used for development, testing and CI when we inject
	// images in the cluster, for example submariner:local, or submariner-route-agent:local
	if spec.Repository == "local" {
		path = componentImage
	} else {
		path = fmt.Sprintf("%s/%s", spec.Repository, componentImage)
	}

	path = fmt.Sprintf("%s:%s", path, spec.Version)
	return path
}
