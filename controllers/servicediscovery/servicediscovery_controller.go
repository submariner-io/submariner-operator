package servicediscovery

import (
	"context"
	goerrors "errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	operatorv1 "github.com/openshift/api/operator/v1"
	operatorclient "github.com/openshift/cluster-dns-operator/pkg/operator/client"
	submarinerv1alpha1 "github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/helpers"
	"github.com/submariner-io/submariner-operator/pkg/images"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_servicediscovery")

const (
	componentName                 = "submariner-lighthouse"
	deploymentName                = "submariner-lighthouse-agent"
	lighthouseCoreDNSName         = "submariner-lighthouse-coredns"
	defaultOpenShiftDNSController = "default"
	lighthouseForwardPluginName   = "lighthouse"
	coreDNSNamespace              = "kube-system"
	coreDNSName                   = "coredns"
)

const (
	serviceDiscoveryImage  = "lighthouse-agent"
	lighthouseCoreDNSImage = "lighthouse-coredns"
)

// NewReconciler returns a new ServiceDiscoveryReconciler
func NewReconciler(mgr manager.Manager) *ServiceDiscoveryReconciler {
	k8sClient, _ := clientset.NewForConfig(mgr.GetConfig())
	operatorClient, _ := operatorclient.NewClient(mgr.GetConfig())
	return &ServiceDiscoveryReconciler{
		client:            mgr.GetClient(),
		log:               ctrl.Log.WithName("controllers").WithName("ServiceDiscovery"),
		scheme:            mgr.GetScheme(),
		k8sClientSet:      k8sClient,
		operatorClientSet: operatorClient}
}

// blank assignment to verify that ServiceDiscoveryReconciler implements reconcile.Reconciler
var _ reconcile.Reconciler = &ServiceDiscoveryReconciler{}

// ServiceDiscoveryReconciler reconciles a ServiceDiscovery object
type ServiceDiscoveryReconciler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client            controllerClient.Client
	log               logr.Logger
	scheme            *runtime.Scheme
	k8sClientSet      clientset.Interface
	operatorClientSet controllerClient.Client
}

// Reconcile reads that state of the cluster for a ServiceDiscovery object and makes changes based on the state read
// and what is in the ServiceDiscovery.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.

// +kubebuilder:rbac:groups=submariner.io,resources=servicediscoveries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=submariner.io,resources=servicediscoveries/status,verbs=get;update;patch
func (r *ServiceDiscoveryReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
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
			deployment := &appsv1.Deployment{}
			opts := []controllerClient.DeleteAllOfOption{
				controllerClient.InNamespace(request.NamespacedName.Namespace),
				controllerClient.MatchingLabels{"app": deploymentName},
			}
			err := r.client.DeleteAllOf(context.TODO(), deployment, opts...)
			return reconcile.Result{}, err
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	lightHouseAgent := newLighthouseAgent(instance)
	if _, err = helpers.ReconcileDeployment(instance, lightHouseAgent, reqLogger,
		r.client, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	lighthouseDnsConfigMap := newLighthouseDnsConfigMap(instance)
	if _, err = helpers.ReconcileConfigMap(instance, lighthouseDnsConfigMap, reqLogger,
		r.client, r.scheme); err != nil {
		log.Error(err, "Error creating the lighthouseCoreDNS configMap")
		return reconcile.Result{}, err
	}

	lighthouseCoreDNSDeployment := newLighthouseCoreDNSDeployment(instance)
	if _, err = helpers.ReconcileDeployment(instance, lighthouseCoreDNSDeployment, reqLogger,
		r.client, r.scheme); err != nil {
		log.Error(err, "Error creating the lighthouseCoreDNS deployment")
		return reconcile.Result{}, err
	}

	lighthouseCoreDNSService := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: lighthouseCoreDNSName, Namespace: instance.Namespace},
		lighthouseCoreDNSService)
	if errors.IsNotFound(err) {
		lighthouseCoreDNSService = newLighthouseCoreDNSService(instance)
		if _, err = helpers.ReconcileService(instance, lighthouseCoreDNSService, reqLogger,
			r.client, r.scheme); err != nil {
			log.Error(err, "Error creating the lighthouseCoreDNS service")
			return reconcile.Result{}, err
		}
	}
	err = updateDNSConfigMap(r.client, r.k8sClientSet, instance, reqLogger)
	if err != nil {
		// Try to update Openshift-DNS
		return reconcile.Result{}, updateOpenshiftClusterDNSOperator(instance, r.client, r.operatorClientSet, reqLogger)
	}

	return reconcile.Result{}, nil
}

func newLighthouseAgent(cr *submarinerv1alpha1.ServiceDiscovery) *appsv1.Deployment {
	replicas := int32(1)
	labels := map[string]string{
		"app":       deploymentName,
		"component": componentName,
	}
	matchLabels := map[string]string{
		"app": deploymentName,
	}

	terminationGracePeriodSeconds := int64(0)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      deploymentName,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "submariner-lighthouse-agent",
							Image:           getImagePath(cr, serviceDiscoveryImage),
							ImagePullPolicy: images.GetPullPolicy(cr.Spec.Version),
							Env: []corev1.EnvVar{
								{Name: "SUBMARINER_NAMESPACE", Value: cr.Spec.Namespace},
								{Name: "SUBMARINER_CLUSTERID", Value: cr.Spec.ClusterID},
								{Name: "SUBMARINER_EXCLUDENS", Value: "submariner,kube-system,operators"},
								{Name: "SUBMARINER_DEBUG", Value: strconv.FormatBool(cr.Spec.Debug)},
								{Name: "SUBMARINER_GLOBALNET_ENABLED", Value: strconv.FormatBool(cr.Spec.GlobalnetEnabled)},
								{Name: "BROKER_K8S_APISERVER", Value: cr.Spec.BrokerK8sApiServer},
								{Name: "BROKER_K8S_APISERVERTOKEN", Value: cr.Spec.BrokerK8sApiServerToken},
								{Name: "BROKER_K8S_REMOTENAMESPACE", Value: cr.Spec.BrokerK8sRemoteNamespace},
								{Name: "BROKER_K8S_CA", Value: cr.Spec.BrokerK8sCA},
							},
						},
					},

					ServiceAccountName:            "submariner-lighthouse",
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
				},
			},
		},
	}
}

func newLighthouseDnsConfigMap(cr *submarinerv1alpha1.ServiceDiscovery) *corev1.ConfigMap {
	labels := map[string]string{
		"app":       lighthouseCoreDNSName,
		"component": componentName,
	}
	config := `{
lighthouse
errors
health
ready
}`
	expectedCorefile := ""
	for _, domain := range append([]string{"clusterset.local"}, cr.Spec.CustomDomains...) {
		expectedCorefile = fmt.Sprintf("%s%s:53 %s\n", expectedCorefile, domain, config)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lighthouseCoreDNSName,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"Corefile": expectedCorefile,
		},
	}
}

func newLighthouseCoreDNSDeployment(cr *submarinerv1alpha1.ServiceDiscovery) *appsv1.Deployment {
	replicas := int32(2)
	labels := map[string]string{
		"app":       lighthouseCoreDNSName,
		"component": componentName,
	}
	matchLabels := map[string]string{
		"app": lighthouseCoreDNSName,
	}

	terminationGracePeriodSeconds := int64(0)
	defaultMode := int32(420)
	allowPrivilegeEscalation := false
	readOnlyRootFilesystem := true

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      lighthouseCoreDNSName,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            lighthouseCoreDNSName,
							Image:           getImagePath(cr, lighthouseCoreDNSImage),
							ImagePullPolicy: images.GetPullPolicy(cr.Spec.Version),
							Args: []string{
								"-conf",
								"/etc/coredns/Corefile",
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "config-volume", MountPath: "/etc/coredns", ReadOnly: true},
							},
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add:  []corev1.Capability{"net_bind_service"},
									Drop: []corev1.Capability{"all"},
								},
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
								ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
							},
						},
					},

					ServiceAccountName:            "submariner-lighthouse",
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Volumes: []corev1.Volume{
						{Name: "config-volume", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: lighthouseCoreDNSName},
							Items: []corev1.KeyToPath{
								{Key: "Corefile", Path: "Corefile"},
							},
							DefaultMode: &defaultMode,
						}}},
					},
				},
			},
		},
	}
}

func newLighthouseCoreDNSService(cr *submarinerv1alpha1.ServiceDiscovery) *corev1.Service {
	labels := map[string]string{
		"app":       lighthouseCoreDNSName,
		"component": componentName,
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      lighthouseCoreDNSName,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:     "udp",
				Protocol: "UDP",
				Port:     53,
				TargetPort: intstr.IntOrString{Type: intstr.Int,
					IntVal: 53},
			}},
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": lighthouseCoreDNSName,
			},
		},
	}
}

func updateDNSConfigMap(client controllerClient.Client, k8sclientSet clientset.Interface, cr *submarinerv1alpha1.ServiceDiscovery,
	reqLogger logr.Logger) error {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		configMap, err := k8sclientSet.CoreV1().ConfigMaps(coreDNSNamespace).Get(coreDNSName, metav1.GetOptions{})
		if err != nil {
			reqLogger.Error(err, "Error retrieving 'coredns' ConfigMap")
			return err
		}

		lighthouseDnsService := &corev1.Service{}
		err = client.Get(context.TODO(), types.NamespacedName{Name: lighthouseCoreDNSName, Namespace: cr.Namespace}, lighthouseDnsService)
		lighthouseClusterIp := lighthouseDnsService.Spec.ClusterIP
		if err != nil || lighthouseClusterIp == "" {
			return goerrors.New("lighthouseDnsService ClusterIp should be available")
		}

		coreFile := configMap.Data["Corefile"]
		newCoreStr := ""
		if strings.Contains(coreFile, "lighthouse-start") {
			// Assume this means we've already set the ConfigMap up, first remove existing lighthouse config
			skip := false
			reqLogger.Info("coredns configmap has lighthouse configuration hence updating")
			lines := strings.Split(coreFile, "\n")
			for _, line := range lines {
				if strings.Contains(line, "lighthouse-start") {
					skip = true
				} else if strings.Contains(line, "lighthouse-end") {
					skip = false
					continue
				}
				if skip {
					continue
				}
				newCoreStr = newCoreStr + line + "\n"
			}
			coreFile = newCoreStr
		} else {
			reqLogger.Info("coredns configmap does not have lighthouse configuration hence creating")
		}
		expectedCorefile := "#lighthouse-start AUTO-GENERATED SECTION. DO NOT EDIT\n"
		for _, domain := range append([]string{"clusterset.local"}, cr.Spec.CustomDomains...) {
			expectedCorefile = fmt.Sprintf("%s%s:53 {\n    forward . %s\n}\n",
				expectedCorefile, domain, lighthouseClusterIp)
		}
		coreFile = expectedCorefile + "#lighthouse-end\n" + coreFile
		log.Info("Updated coredns ConfigMap " + coreFile)
		configMap.Data["Corefile"] = coreFile
		// Potentially retried
		_, err = k8sclientSet.CoreV1().ConfigMaps(coreDNSNamespace).Update(configMap)
		return err
	})
	return retryErr
}

func updateOpenshiftClusterDNSOperator(instance *submarinerv1alpha1.ServiceDiscovery, client, operatorClient controllerClient.Client,
	reqLogger logr.Logger) error {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dnsOperator := &operatorv1.DNS{}
		if err := client.Get(context.TODO(), types.NamespacedName{Name: defaultOpenShiftDNSController}, dnsOperator); err != nil {
			return err
		}

		lighthouseDnsService := &corev1.Service{}
		err := operatorClient.Get(context.TODO(), types.NamespacedName{Name: lighthouseCoreDNSName, Namespace: instance.Namespace},
			lighthouseDnsService)
		if err != nil || lighthouseDnsService.Spec.ClusterIP == "" {
			return goerrors.New("lighthouseDnsService ClusterIp should be available")
		}

		var updatedForwardServers []operatorv1.Server
		changed := false
		containsLighthouse := false
		lighthouseDomains := append([]string{"clusterset.local"}, instance.Spec.CustomDomains...)
		existingDomains := make([]string, 0)
		for _, forwardServer := range dnsOperator.Spec.Servers {
			if forwardServer.Name == lighthouseForwardPluginName {
				containsLighthouse = true
				existingDomains = append(existingDomains, forwardServer.Zones...)
				for _, upstreams := range forwardServer.ForwardPlugin.Upstreams {
					if upstreams != lighthouseDnsService.Spec.ClusterIP {
						changed = true
					}
				}
				if changed {
					continue
				}
			} else {
				updatedForwardServers = append(updatedForwardServers, forwardServer)
			}
		}

		sort.Strings(lighthouseDomains)
		sort.Strings(existingDomains)
		if !reflect.DeepEqual(lighthouseDomains, existingDomains) {
			changed = true
			reqLogger.Info(fmt.Sprintf("Configured lighthouse zones changed from %v to %v", existingDomains, lighthouseDomains))
		}
		if containsLighthouse && !changed {
			reqLogger.Info("Forward plugin is already configured in Cluster DNS Operator CR")
			return nil
		}
		reqLogger.Info("Lighthouse DNS configuration changed, hence updating Cluster DNS Operator CR")

		for _, domain := range lighthouseDomains {
			lighthouseServer := operatorv1.Server{
				Name:  lighthouseForwardPluginName,
				Zones: []string{domain},
				ForwardPlugin: operatorv1.ForwardPlugin{
					Upstreams: []string{lighthouseDnsService.Spec.ClusterIP},
				},
			}
			updatedForwardServers = append(updatedForwardServers, lighthouseServer)
		}

		dnsOperator.Spec.Servers = updatedForwardServers

		toUpdate := &operatorv1.DNS{ObjectMeta: metav1.ObjectMeta{
			Name:   dnsOperator.Name,
			Labels: dnsOperator.Labels,
		}}

		result, err := controllerutil.CreateOrUpdate(context.TODO(), client, toUpdate, func() error {
			toUpdate.Spec = dnsOperator.Spec
			for k, v := range dnsOperator.Labels {
				toUpdate.Labels[k] = v
			}
			return nil
		})

		if result == controllerutil.OperationResultUpdated {
			reqLogger.Info("Updated Cluster DNS Operator", "DnsOperator.Name", dnsOperator.Name)
		}
		return err
	})
	return retryErr
}

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

func (r *ServiceDiscoveryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// These are required so that we can manipulate DNS ConfigMap
	if err := operatorv1.Install(mgr.GetScheme()); err != nil {
		return err
	}
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

	// Watch for changes to secondary resource Deployment and requeue the owner ServiceDiscovery
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &submarinerv1alpha1.ServiceDiscovery{},
	})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&submarinerv1alpha1.ServiceDiscovery{}).
		Complete(r)
}
