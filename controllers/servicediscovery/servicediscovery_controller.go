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
	"github.com/submariner-io/submariner-operator/controllers/metrics"
	"github.com/submariner-io/submariner-operator/pkg/images"
	"github.com/submariner-io/submariner-operator/pkg/names"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
	defaultCoreDNSNamespace       = "kube-system"
	coreDNSName                   = "coredns"
)

// NewReconciler returns a new ServiceDiscoveryReconciler
func NewReconciler(mgr manager.Manager) *ServiceDiscoveryReconciler {
	k8sClient, _ := clientset.NewForConfig(mgr.GetConfig())
	operatorClient, _ := operatorclient.NewClient(mgr.GetConfig())
	return &ServiceDiscoveryReconciler{
		client:            mgr.GetClient(),
		config:            mgr.GetConfig(),
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
	config            *rest.Config
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
func (r *ServiceDiscoveryReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ServiceDiscovery")

	// Fetch the ServiceDiscovery instance
	instance := &submarinerv1alpha1.ServiceDiscovery{}

	err := r.client.Get(ctx, request.NamespacedName, instance)
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
			err := r.client.DeleteAllOf(ctx, deployment, opts...)
			return reconcile.Result{}, err
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.ObjectMeta.DeletionTimestamp != nil {
		// Graceful deletion has been requested, ignore the object
		return reconcile.Result{}, nil
	}

	lightHouseAgent := newLighthouseAgent(instance)
	if _, err = helpers.ReconcileDeployment(instance, lightHouseAgent, reqLogger,
		r.client, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	err = metrics.Setup(instance.Namespace, instance, lightHouseAgent.GetLabels(), 8082, r.client, r.config, r.scheme, reqLogger)
	if err != nil {
		return reconcile.Result{}, err
	}

	lighthouseDNSConfigMap := newLighthouseDNSConfigMap(instance)
	if _, err = helpers.ReconcileConfigMap(instance, lighthouseDNSConfigMap, reqLogger,
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
	err = r.client.Get(ctx, types.NamespacedName{Name: lighthouseCoreDNSName, Namespace: instance.Namespace},
		lighthouseCoreDNSService)
	if errors.IsNotFound(err) {
		lighthouseCoreDNSService = newLighthouseCoreDNSService(instance)
		if _, err = helpers.ReconcileService(instance, lighthouseCoreDNSService, reqLogger,
			r.client, r.scheme); err != nil {
			log.Error(err, "Error creating the lighthouseCoreDNS service")
			return reconcile.Result{}, err
		}
	}
	err = metrics.Setup(instance.Namespace, instance, lighthouseCoreDNSDeployment.GetLabels(), 9153, r.client, r.config, r.scheme, reqLogger)
	if err != nil {
		return reconcile.Result{}, err
	}
	if instance.Spec.CoreDNSCustomConfig != nil && instance.Spec.CoreDNSCustomConfig.ConfigMapName != "" {
		err = updateDNSCustomConfigMap(ctx, r.client, r.k8sClientSet, instance, reqLogger)
		if err != nil {
			reqLogger.Error(err, "Error updating the 'custom-coredns' ConfigMap")
			return reconcile.Result{}, err
		}
	} else {
		err = updateDNSConfigMap(ctx, r.client, r.k8sClientSet, instance, reqLogger)
	}

	if errors.IsNotFound(err) {
		// Try to update Openshift-DNS
		return reconcile.Result{}, updateOpenshiftClusterDNSOperator(ctx, instance, r.client, r.operatorClientSet, reqLogger)
	} else if err != nil {
		reqLogger.Error(err, "Error updating the 'coredns' ConfigMap")
		return reconcile.Result{}, err
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
							Image:           getImagePath(cr, names.ServiceDiscoveryImage, names.ServiceDiscoveryComponent),
							ImagePullPolicy: helpers.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.ServiceDiscoveryComponent]),
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

					ServiceAccountName:            "submariner-lighthouse-agent",
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
				},
			},
		},
	}
}

func newLighthouseDNSConfigMap(cr *submarinerv1alpha1.ServiceDiscovery) *corev1.ConfigMap {
	labels := map[string]string{
		"app":       lighthouseCoreDNSName,
		"component": componentName,
	}
	config := `{
lighthouse
errors
health
ready
prometheus :9153
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

func newCoreDNSCustomConfigMap(cr *submarinerv1alpha1.ServiceDiscovery) *corev1.ConfigMap {
	namespace := defaultCoreDNSNamespace
	if cr.Spec.CoreDNSCustomConfig.Namespace != "" {
		namespace = cr.Spec.CoreDNSCustomConfig.Namespace
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Spec.CoreDNSCustomConfig.ConfigMapName,
			Namespace: namespace,
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
							Image:           getImagePath(cr, names.LighthouseCoreDNSImage, names.LighthouseCoreDNSComponent),
							ImagePullPolicy: helpers.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.LighthouseCoreDNSComponent]),
							Env: []corev1.EnvVar{
								{Name: "SUBMARINER_CLUSTERID", Value: cr.Spec.ClusterID},
							},
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

					ServiceAccountName:            "submariner-lighthouse-coredns",
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

func updateDNSCustomConfigMap(ctx context.Context, client controllerClient.Client, k8sclientSet clientset.Interface,
	cr *submarinerv1alpha1.ServiceDiscovery, reqLogger logr.Logger) error {
	var configFunc func(context.Context, *corev1.ConfigMap) (*corev1.ConfigMap, error)
	customCoreDNSName := cr.Spec.CoreDNSCustomConfig.ConfigMapName
	coreDNSNamespace := defaultCoreDNSNamespace
	if cr.Spec.CoreDNSCustomConfig.Namespace != "" {
		coreDNSNamespace = cr.Spec.CoreDNSCustomConfig.Namespace
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		configMap, err := k8sclientSet.CoreV1().ConfigMaps(coreDNSNamespace).Get(ctx, customCoreDNSName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			configFunc = func(ctx context.Context, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
				return k8sclientSet.CoreV1().ConfigMaps(coreDNSNamespace).Create(ctx, cm, metav1.CreateOptions{})
			}
			configMap = newCoreDNSCustomConfigMap(cr)
		} else if err != nil {
			return err
		} else {
			configFunc = func(ctx context.Context, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
				return k8sclientSet.CoreV1().ConfigMaps(coreDNSNamespace).Update(ctx, cm, metav1.UpdateOptions{})
			}
		}

		lighthouseDNSService := &corev1.Service{}
		err = client.Get(ctx, types.NamespacedName{Name: lighthouseCoreDNSName, Namespace: cr.Namespace}, lighthouseDNSService)
		lighthouseClusterIP := lighthouseDNSService.Spec.ClusterIP
		if err != nil || lighthouseClusterIP == "" {
			return goerrors.New("lighthouseDNSService ClusterIp should be available")
		}

		if configMap.Data == nil {
			reqLogger.Info("Initializing configMap.Data in " + customCoreDNSName)
			configMap.Data = make(map[string]string)
		}

		if _, ok := configMap.Data["lighthouse.server"]; ok {
			reqLogger.Info("Overwriting existing lighthouse.server data in " + customCoreDNSName)
		}

		coreFile := ""
		for _, domain := range append([]string{"clusterset.local"}, cr.Spec.CustomDomains...) {
			coreFile = fmt.Sprintf("%s%s:53 {\n    forward . %s\n}\n",
				coreFile, domain, lighthouseClusterIP)
		}
		log.Info("Updating coredns-custom ConfigMap for lighthouse.server: " + coreFile)
		configMap.Data["lighthouse.server"] = coreFile
		// Potentially retried
		_, err = configFunc(ctx, configMap)
		return err
	})
	return retryErr
}

func updateDNSConfigMap(ctx context.Context, client controllerClient.Client, k8sclientSet clientset.Interface,
	cr *submarinerv1alpha1.ServiceDiscovery, reqLogger logr.Logger) error {
	coreDNSNamespace := defaultCoreDNSNamespace
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		configMap, err := k8sclientSet.CoreV1().ConfigMaps(coreDNSNamespace).Get(ctx, coreDNSName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		lighthouseDNSService := &corev1.Service{}
		err = client.Get(ctx, types.NamespacedName{Name: lighthouseCoreDNSName, Namespace: cr.Namespace}, lighthouseDNSService)
		lighthouseClusterIP := lighthouseDNSService.Spec.ClusterIP
		if err != nil || lighthouseClusterIP == "" {
			return goerrors.New("lighthouseDNSService ClusterIp should be available")
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
				expectedCorefile, domain, lighthouseClusterIP)
		}
		coreFile = expectedCorefile + "#lighthouse-end\n" + coreFile
		log.Info("Updated coredns ConfigMap " + coreFile)
		configMap.Data["Corefile"] = coreFile
		// Potentially retried
		_, err = k8sclientSet.CoreV1().ConfigMaps(coreDNSNamespace).Update(ctx, configMap, metav1.UpdateOptions{})
		return err
	})
	return retryErr
}

func updateOpenshiftClusterDNSOperator(ctx context.Context, instance *submarinerv1alpha1.ServiceDiscovery,
	client, operatorClient controllerClient.Client, reqLogger logr.Logger) error {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dnsOperator := &operatorv1.DNS{}
		if err := client.Get(ctx, types.NamespacedName{Name: defaultOpenShiftDNSController}, dnsOperator); err != nil {
			return err
		}

		lighthouseDNSService := &corev1.Service{}
		err := operatorClient.Get(ctx, types.NamespacedName{Name: lighthouseCoreDNSName, Namespace: instance.Namespace},
			lighthouseDNSService)
		if err != nil || lighthouseDNSService.Spec.ClusterIP == "" {
			return goerrors.New("lighthouseDNSService ClusterIp should be available")
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
					if upstreams != lighthouseDNSService.Spec.ClusterIP {
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
					Upstreams: []string{lighthouseDNSService.Spec.ClusterIP},
				},
			}
			updatedForwardServers = append(updatedForwardServers, lighthouseServer)
		}

		dnsOperator.Spec.Servers = updatedForwardServers

		toUpdate := &operatorv1.DNS{ObjectMeta: metav1.ObjectMeta{
			Name:   dnsOperator.Name,
			Labels: dnsOperator.Labels,
		}}

		result, err := controllerutil.CreateOrUpdate(ctx, client, toUpdate, func() error {
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

func getImagePath(submariner *submarinerv1alpha1.ServiceDiscovery, imageName, componentName string) string {
	return images.GetImagePath(submariner.Spec.Repository, submariner.Spec.Version, imageName, componentName,
		submariner.Spec.ImageOverrides)
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
