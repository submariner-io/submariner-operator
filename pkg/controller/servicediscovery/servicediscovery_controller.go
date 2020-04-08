package servicediscovery

import (
	"context"
	"fmt"

	submarinerv1alpha1 "github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_servicediscovery")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new ServiceDiscovery Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileServiceDiscovery{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("servicediscovery-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ServiceDiscovery
	err = c.Watch(&source.Kind{Type: &submarinerv1alpha1.ServiceDiscovery{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner ServiceDiscovery
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &submarinerv1alpha1.ServiceDiscovery{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileServiceDiscovery implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileServiceDiscovery{}

// ReconcileServiceDiscovery reconciles a ServiceDiscovery object
type ReconcileServiceDiscovery struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a ServiceDiscovery object and makes changes based on the state read
// and what is in the ServiceDiscovery.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileServiceDiscovery) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ServiceDiscovery")

	// Fetch the ServiceDiscovery instance
	instance := &submarinerv1alpha1.ServiceDiscovery{}
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

	// Create Lighthouse-agent
	lighthouseAgent := newLighthouseAgent(instance)

	// Set ServiceDiscovery instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, lighthouseAgent, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this lighthouse-agent already exists
	found := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: lighthouseAgent.Name, Namespace: lighthouseAgent.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new lighthouse-agent", "Pod.Namespace", lighthouseAgent.Namespace, "Pod.Name", lighthouseAgent.Name)
		err = r.client.Create(context.TODO(), lighthouseAgent)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Lighthouse-agent created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// lighthouse-agent already exists - don't requeue
	reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
	return reconcile.Result{}, nil
}

func newLighthouseAgent(cr *submarinerv1alpha1.ServiceDiscovery) *appsv1.DaemonSet {
	labels := map[string]string{
		"app":       "submariner-lighthouse-agent",
		"component": "lighthouse-agent",
	}

	matchLabels := map[string]string{
		"app": "submariner-lighthouse-agent",
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

	serviceDiscoveryDaemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      "submariner-lighthouse-agent",
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
							Name:            "submariner-lighthouse-agent",
							Image:           getImagePath(cr, serviceDiscoveryImage),
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

	return serviceDiscoveryDaemonSet
}

const (
	serviceDiscoveryImage  = "lighthouse-agent"
)

func getImagePath(submariner *submarinerv1alpha1.ServiceDiscovery, componentImage string) string {
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
