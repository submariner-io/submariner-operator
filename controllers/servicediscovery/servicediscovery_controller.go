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
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/finalizer"
	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/admiral/pkg/syncer/broker"
	"github.com/submariner-io/admiral/pkg/util"
	submarinerv1alpha1 "github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/apply"
	"github.com/submariner-io/submariner-operator/controllers/metrics"
	"github.com/submariner-io/submariner-operator/pkg/images"
	"github.com/submariner-io/submariner-operator/pkg/names"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logf.Log.WithName("controller_servicediscovery")

const (
	componentName                 = "submariner-lighthouse"
	defaultOpenShiftDNSController = "default"
	lighthouseForwardPluginName   = "lighthouse"
	defaultCoreDNSNamespace       = "kube-system"
	coreDNSName                   = "coredns"
	microshiftDNSNamespace        = "openshift-dns"
	microshiftDNSConfigMap        = "dns-default"
	coreDNSDefaultPort            = "53"
)

// Reconciler reconciles a ServiceDiscovery object.
type Reconciler struct {
	// This client is scoped to the operator namespace intended to only be used for resources created and maintained by this
	// controller. Also it's a split client that reads objects from the cache and writes to the apiserver.
	ScopedClient controllerClient.Client
	// This client can be used to access any other resource not in the operator namespace.
	GeneralClient controllerClient.Client
	Scheme        *runtime.Scheme
	RestConfig    *rest.Config
}

// blank assignment to verify that Reconciler implements reconcile.Reconciler.
var _ reconcile.Reconciler = &Reconciler{}

//+kubebuilder:rbac:groups=submariner.io,resources=servicediscoveries,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=submariner.io,resources=servicediscoveries/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=submariner.io,resources=servicediscoveries/finalizers,verbs=update

// Reconcile reads that state of the cluster for a ServiceDiscovery object and makes changes based on the state read
// and what is in the ServiceDiscovery.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.V(2).WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ServiceDiscovery")

	instance, err := r.getServiceDiscovery(ctx, request.NamespacedName)
	if apierrors.IsNotFound(err) {
		// Request object not found, could have been deleted after reconcile request.
		// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
		// Return and don't requeue
		deployment := &appsv1.Deployment{}
		opts := []controllerClient.DeleteAllOfOption{
			controllerClient.InNamespace(request.NamespacedName.Namespace),
			controllerClient.MatchingLabels{"app": names.ServiceDiscoveryComponent},
		}
		err := r.ScopedClient.DeleteAllOf(ctx, deployment, opts...)

		return reconcile.Result{}, errors.Wrap(err, "error deleting resource")
	}

	if err != nil {
		return reconcile.Result{}, err
	}

	instance, err = r.addFinalizer(ctx, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.GetDeletionTimestamp().IsZero() {
		log.Info("ServiceDiscovery is being deleted")
		return r.doCleanup(ctx, instance)
	}

	err = r.ensureLightHouseAgent(ctx, instance, reqLogger)
	if err != nil {
		return reconcile.Result{}, err
	}

	lighthouseDNSConfigMap := newLighthouseDNSConfigMap(instance)
	if _, err = apply.ConfigMap(ctx, instance, lighthouseDNSConfigMap, reqLogger,
		r.ScopedClient, r.Scheme); err != nil {
		log.Error(err, "Error creating the lighthouseCoreDNS configMap")
		return reconcile.Result{}, errors.Wrap(err, "error reconciling ConfigMap")
	}

	err = r.ensureLighthouseCoreDNSDeployment(ctx, instance, reqLogger)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.ensureLighthouseCoreDNSService(ctx, instance, reqLogger)
	if err != nil {
		return reconcile.Result{}, err
	}

	if instance.Spec.CoreDNSCustomConfig != nil && instance.Spec.CoreDNSCustomConfig.ConfigMapName != "" {
		err = r.updateDNSCustomConfigMap(ctx, instance, reqLogger)
		if err != nil {
			reqLogger.Error(err, "Error updating the 'custom-coredns' ConfigMap")
			return reconcile.Result{}, err
		}
	} else {
		err = r.configureDNSConfigMap(ctx, instance, defaultCoreDNSNamespace, coreDNSName)
	}

	if apierrors.IsNotFound(err) {
		// Try to update Openshift-DNS
		return reconcile.Result{}, r.configureOpenshiftClusterDNSOperator(ctx, instance)
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) getServiceDiscovery(ctx context.Context, key types.NamespacedName) (*submarinerv1alpha1.ServiceDiscovery, error) {
	instance := &submarinerv1alpha1.ServiceDiscovery{}

	err := r.ScopedClient.Get(ctx, key, instance)
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving ServiceDiscovery resource")
	}

	return instance, nil
}

func (r *Reconciler) addFinalizer(ctx context.Context,
	instance *submarinerv1alpha1.ServiceDiscovery,
) (*submarinerv1alpha1.ServiceDiscovery, error) {
	added, err := finalizer.Add(ctx, resource.ForControllerClient(r.ScopedClient, instance.Namespace,
		&submarinerv1alpha1.ServiceDiscovery{}), instance, names.CleanupFinalizer)
	if err != nil {
		return nil, err //nolint:wrapcheck // No need to wrap
	}

	if !added {
		return instance, nil
	}

	return r.getServiceDiscovery(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name})
}

func newLighthouseAgent(cr *submarinerv1alpha1.ServiceDiscovery, name string) *appsv1.Deployment {
	labels := map[string]string{
		"app":       name,
		"component": componentName,
	}
	matchLabels := map[string]string{
		"app": name,
	}

	volumeMounts := []corev1.VolumeMount{}
	volumes := []corev1.Volume{}

	if cr.Spec.BrokerK8sSecret != "" {
		// We've got a secret, mount it where the syncer expects it
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "brokersecret",
			MountPath: broker.SecretPath(cr.Spec.BrokerK8sSecret),
			ReadOnly:  true,
		})

		volumes = append(volumes, corev1.Volume{
			Name:         "brokersecret",
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: cr.Spec.BrokerK8sSecret}},
		})
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
			Replicas: pointer.Int32(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           getImagePath(cr, names.ServiceDiscoveryImage, names.ServiceDiscoveryComponent),
							ImagePullPolicy: images.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.ServiceDiscoveryComponent]),
							Env: []corev1.EnvVar{
								{Name: "SUBMARINER_NAMESPACE", Value: cr.Spec.Namespace},
								{Name: "SUBMARINER_CLUSTERID", Value: cr.Spec.ClusterID},
								{Name: "SUBMARINER_EXCLUDENS", Value: "submariner,kube-system,operators"},
								{Name: "SUBMARINER_DEBUG", Value: strconv.FormatBool(cr.Spec.Debug)},
								{Name: "SUBMARINER_GLOBALNET_ENABLED", Value: strconv.FormatBool(cr.Spec.GlobalnetEnabled)},
								{Name: broker.EnvironmentVariable("ApiServer"), Value: cr.Spec.BrokerK8sApiServer},
								{Name: broker.EnvironmentVariable("ApiServerToken"), Value: cr.Spec.BrokerK8sApiServerToken},
								{Name: broker.EnvironmentVariable("RemoteNamespace"), Value: cr.Spec.BrokerK8sRemoteNamespace},
								{Name: broker.EnvironmentVariable("CA"), Value: cr.Spec.BrokerK8sCA},
								{Name: broker.EnvironmentVariable("Insecure"), Value: strconv.FormatBool(cr.Spec.BrokerK8sInsecure)},
								{Name: broker.EnvironmentVariable("Secret"), Value: cr.Spec.BrokerK8sSecret},
							},
							VolumeMounts: volumeMounts,
						},
					},

					ServiceAccountName:            "submariner-lighthouse-agent",
					Tolerations:                   cr.Spec.Tolerations,
					NodeSelector:                  cr.Spec.NodeSelector,
					TerminationGracePeriodSeconds: pointer.Int64(0),
					Volumes:                       volumes,
				},
			},
		},
	}
}

func newLighthouseDNSConfigMap(cr *submarinerv1alpha1.ServiceDiscovery) *corev1.ConfigMap {
	labels := map[string]string{
		"app":       names.LighthouseCoreDNSComponent,
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
			Name:      names.LighthouseCoreDNSComponent,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"Corefile": expectedCorefile,
		},
	}
}

func newCoreDNSCustomConfigMap(config *submarinerv1alpha1.CoreDNSCustomConfig) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.ConfigMapName,
			Namespace: getCustomCoreDNSNamespace(config),
		},
	}
}

func newLighthouseCoreDNSDeployment(cr *submarinerv1alpha1.ServiceDiscovery) *appsv1.Deployment {
	labels := map[string]string{
		"app":       names.LighthouseCoreDNSComponent,
		"component": componentName,
	}
	matchLabels := map[string]string{
		"app": names.LighthouseCoreDNSComponent,
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      names.LighthouseCoreDNSComponent,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
			Replicas: pointer.Int32(2),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            names.LighthouseCoreDNSComponent,
							Image:           getImagePath(cr, names.LighthouseCoreDNSImage, names.LighthouseCoreDNSComponent),
							ImagePullPolicy: images.GetPullPolicy(cr.Spec.Version, cr.Spec.ImageOverrides[names.LighthouseCoreDNSComponent]),
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
								AllowPrivilegeEscalation: pointer.Bool(false),
								ReadOnlyRootFilesystem:   pointer.Bool(true),
							},
						},
					},

					ServiceAccountName:            "submariner-lighthouse-coredns",
					TerminationGracePeriodSeconds: pointer.Int64(0),
					Tolerations:                   cr.Spec.Tolerations,
					NodeSelector:                  cr.Spec.NodeSelector,
					Volumes: []corev1.Volume{
						{Name: "config-volume", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: names.LighthouseCoreDNSComponent},
							Items: []corev1.KeyToPath{
								{Key: "Corefile", Path: "Corefile"},
							},
							DefaultMode: pointer.Int32(420),
						}}},
					},
				},
			},
		},
	}
}

func newLighthouseCoreDNSService(cr *submarinerv1alpha1.ServiceDiscovery) *corev1.Service {
	labels := map[string]string{
		"app":       names.LighthouseCoreDNSComponent,
		"component": componentName,
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      names.LighthouseCoreDNSComponent,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:     "udp",
				Protocol: "UDP",
				Port:     53,
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 53,
				},
			}},
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": names.LighthouseCoreDNSComponent,
			},
		},
	}
}

func getCustomCoreDNSNamespace(config *submarinerv1alpha1.CoreDNSCustomConfig) string {
	if config.Namespace != "" {
		return config.Namespace
	}

	return defaultCoreDNSNamespace
}

func (r *Reconciler) updateDNSCustomConfigMap(ctx context.Context, cr *submarinerv1alpha1.ServiceDiscovery,
	reqLogger logr.Logger,
) error {
	configMap := newCoreDNSCustomConfigMap(cr.Spec.CoreDNSCustomConfig)

	_, err := controllerutil.CreateOrUpdate(ctx, r.GeneralClient, configMap, func() error {
		lighthouseDNSService := &corev1.Service{}
		err := r.ScopedClient.Get(ctx, types.NamespacedName{Name: names.LighthouseCoreDNSComponent, Namespace: cr.Namespace},
			lighthouseDNSService)
		lighthouseClusterIP := lighthouseDNSService.Spec.ClusterIP
		if err != nil || lighthouseClusterIP == "" {
			return goerrors.New("lighthouseDNSService ClusterIp should be available")
		}

		if configMap.Data == nil {
			reqLogger.Info("Initializing configMap.Data in " + configMap.Name)
			configMap.Data = make(map[string]string)
		}

		if _, ok := configMap.Data["lighthouse.server"]; ok {
			reqLogger.Info("Overwriting existing lighthouse.server data in " + configMap.Name)
		}

		coreFile := ""
		for _, domain := range append([]string{"clusterset.local"}, cr.Spec.CustomDomains...) {
			coreFile = fmt.Sprintf("%s%s:53 {\n    forward . %s\n}\n",
				coreFile, domain, lighthouseClusterIP)
		}

		log.Info("Updating coredns-custom ConfigMap for lighthouse.server: " + coreFile)
		configMap.Data["lighthouse.server"] = coreFile

		return nil
	})

	return errors.Wrap(err, "error updating DNS custom ConfigMap")
}

func (r *Reconciler) configureDNSConfigMap(ctx context.Context, cr *submarinerv1alpha1.ServiceDiscovery, configMapNamespace,
	configMapName string,
) error {
	lighthouseDNSService := &corev1.Service{}

	err := r.ScopedClient.Get(ctx, types.NamespacedName{Name: names.LighthouseCoreDNSComponent, Namespace: cr.Namespace},
		lighthouseDNSService)
	if err != nil {
		return errors.Wrap(err, "error retrieving lighthouse DNS Service")
	}

	if lighthouseDNSService.Spec.ClusterIP == "" {
		return goerrors.New("the lighthouse DNS Service ClusterIP is not set")
	}

	return r.updateLighthouseConfigInConfigMap(ctx, cr, configMapNamespace, configMapName, lighthouseDNSService.Spec.ClusterIP)
}

func (r *Reconciler) updateLighthouseConfigInConfigMap(ctx context.Context, cr *submarinerv1alpha1.ServiceDiscovery,
	configMapNamespace, configMapName, clusterIP string,
) error {
	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: configMapNamespace, Name: configMapName}}
	err := util.MustUpdate(ctx, resource.ForControllerClient(r.GeneralClient, configMap.Namespace, configMap), configMap,
		func(obj runtime.Object) (runtime.Object, error) {
			existing := obj.(*corev1.ConfigMap)

			coreFile := existing.Data["Corefile"]
			if strings.Contains(coreFile, "lighthouse-start") {
				// Assume this means we've already set the ConfigMap up, first remove existing lighthouse config
				newCoreStr := ""
				skip := false
				log.Info("coredns configmap has lighthouse configuration hence updating")
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
				log.Info("coredns configmap does not have lighthouse configuration hence creating")
			}

			if clusterIP != "" {
				coreDNSPort := findCoreDNSListeningPort(coreFile)

				expectedCorefile := "#lighthouse-start AUTO-GENERATED SECTION. DO NOT EDIT\n"
				for _, domain := range append([]string{"clusterset.local"}, cr.Spec.CustomDomains...) {
					expectedCorefile = fmt.Sprintf("%s%s:%s {\n    forward . %s\n}\n",
						expectedCorefile, domain, coreDNSPort, clusterIP)
				}

				coreFile = expectedCorefile + "#lighthouse-end\n" + coreFile
			}

			log.Info("Updated coredns ConfigMap " + coreFile)
			existing.Data["Corefile"] = coreFile

			return existing, nil
		})

	return errors.Wrap(err, "error updating DNS ConfigMap")
}

func findCoreDNSListeningPort(coreFile string) string {
	coreDNSPort := coreDNSDefaultPort
	coreDNSPortRegex := regexp.MustCompile(`\.:(\d*?)\s*{`)

	matches := coreDNSPortRegex.FindStringSubmatch(coreFile)
	if len(matches) == 2 {
		coreDNSPort = matches[1]
	}

	return coreDNSPort
}

func (r *Reconciler) configureOpenshiftClusterDNSOperator(ctx context.Context, instance *submarinerv1alpha1.ServiceDiscovery) error {
	lighthouseDNSService := &corev1.Service{}

	err := r.ScopedClient.Get(ctx, types.NamespacedName{Name: names.LighthouseCoreDNSComponent, Namespace: instance.Namespace},
		lighthouseDNSService)
	if err != nil {
		return errors.Wrap(err, "error retrieving lighthouse DNS Service")
	}

	if lighthouseDNSService.Spec.ClusterIP == "" {
		return goerrors.New("the lighthouse DNS Service ClusterIP is not set")
	}

	return r.updateLighthouseConfigInOpenshiftDNSOperator(ctx, instance, lighthouseDNSService.Spec.ClusterIP)
}

func (r *Reconciler) updateLighthouseConfigInOpenshiftDNSOperator(ctx context.Context, instance *submarinerv1alpha1.ServiceDiscovery,
	clusterIP string,
) error {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dnsOperator := &operatorv1.DNS{}
		if err := r.GeneralClient.Get(ctx, types.NamespacedName{Name: defaultOpenShiftDNSController}, dnsOperator); err != nil {
			// microshift uses the coredns image, but the DNS operator and CRDs are off
			if meta.IsNoMatchError(err) {
				err = r.configureDNSConfigMap(ctx, instance, microshiftDNSNamespace, microshiftDNSConfigMap)
				return errors.Wrapf(err, "error trying to update microshift coredns configmap %q in namespace %q",
					microshiftDNSNamespace, microshiftDNSNamespace)
			}

			return err
		}

		updatedForwardServers := getUpdatedForwardServers(instance, dnsOperator, clusterIP)
		if updatedForwardServers == nil {
			return nil
		}

		dnsOperator.Spec.Servers = updatedForwardServers

		err := util.MustUpdate(ctx, resource.ForControllerClient(r.GeneralClient, "", dnsOperator), dnsOperator,
			func(obj runtime.Object) (runtime.Object, error) {
				existing := obj.(*operatorv1.DNS)
				existing.Spec = dnsOperator.Spec

				for k, v := range dnsOperator.Labels {
					existing.Labels[k] = v
				}

				return obj, nil
			})

		if err == nil {
			log.Info("Updated Cluster DNS Operator", "DnsOperator.Name", dnsOperator.Name)
		}
		return err
	})

	return errors.Wrap(retryErr, "error updating Openshift DNS operator")
}

func getUpdatedForwardServers(instance *submarinerv1alpha1.ServiceDiscovery, dnsOperator *operatorv1.DNS,
	clusterIP string,
) []operatorv1.Server {
	updatedForwardServers := make([]operatorv1.Server, 0)
	changed := false
	containsLighthouse := false
	existingDomains := make([]string, 0)

	lighthouseDomains := append([]string{"clusterset.local"}, instance.Spec.CustomDomains...)

	for _, forwardServer := range dnsOperator.Spec.Servers {
		if forwardServer.Name == lighthouseForwardPluginName {
			containsLighthouse = true

			existingDomains = append(existingDomains, forwardServer.Zones...)

			for _, upstreams := range forwardServer.ForwardPlugin.Upstreams {
				if upstreams != clusterIP {
					changed = true
				}
			}
		} else {
			updatedForwardServers = append(updatedForwardServers, forwardServer)
		}
	}

	if clusterIP == "" {
		return updatedForwardServers
	}

	sort.Strings(lighthouseDomains)
	sort.Strings(existingDomains)

	if !reflect.DeepEqual(lighthouseDomains, existingDomains) {
		changed = true

		log.Info(fmt.Sprintf("Configured lighthouse zones changed from %v to %v", existingDomains, lighthouseDomains))
	}

	if containsLighthouse && !changed {
		log.Info("Forward plugin is already configured in Cluster DNS Operator CR")
		return nil
	}

	log.Info("Lighthouse DNS configuration changed, hence updating Cluster DNS Operator CR")

	for _, domain := range lighthouseDomains {
		lighthouseServer := operatorv1.Server{
			Name:  lighthouseForwardPluginName,
			Zones: []string{domain},
			ForwardPlugin: operatorv1.ForwardPlugin{
				Upstreams: []string{clusterIP},
			},
		}
		updatedForwardServers = append(updatedForwardServers, lighthouseServer)
	}

	return updatedForwardServers
}

func getImagePath(submariner *submarinerv1alpha1.ServiceDiscovery, imageName, componentName string) string {
	return images.GetImagePath(submariner.Spec.Repository, submariner.Spec.Version, imageName, componentName,
		submariner.Spec.ImageOverrides)
}

//nolint:wrapcheck // No need to wrap errors here.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	// These are required so that we can manipulate DNS ConfigMap
	if err := operatorv1.Install(mgr.GetScheme()); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named("servicediscovery-controller").
		// Watch for changes to primary resource ServiceDiscovery
		For(&submarinerv1alpha1.ServiceDiscovery{}).
		// Watch for changes to secondary resource Deployment and requeue the owner ServiceDiscovery
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *Reconciler) ensureLightHouseAgent(ctx context.Context, instance *submarinerv1alpha1.ServiceDiscovery, reqLogger logr.Logger,
) error {
	lightHouseAgent := newLighthouseAgent(instance, names.ServiceDiscoveryComponent)
	if _, err := apply.Deployment(ctx, instance, lightHouseAgent, reqLogger,
		r.ScopedClient, r.Scheme); err != nil {
		return errors.Wrap(err, "error reconciling agent deployment")
	}

	err := metrics.Setup(ctx, names.ServiceDiscoveryComponent, instance.Namespace, "app", names.ServiceDiscoveryComponent,
		instance, 8082, r.ScopedClient, r.RestConfig, r.Scheme, reqLogger)
	if err != nil {
		return errors.Wrap(err, "error setting up metrics")
	}

	return nil
}

func (r *Reconciler) ensureLighthouseCoreDNSDeployment(ctx context.Context, instance *submarinerv1alpha1.ServiceDiscovery,
	reqLogger logr.Logger,
) error {
	lighthouseCoreDNSDeployment := newLighthouseCoreDNSDeployment(instance)
	if _, err := apply.Deployment(ctx, instance, lighthouseCoreDNSDeployment, reqLogger,
		r.ScopedClient, r.Scheme); err != nil {
		log.Error(err, "Error creating the lighthouseCoreDNS deployment")
		return errors.Wrap(err, "error reconciling coredns deployment")
	}

	err := metrics.Setup(ctx, names.LighthouseCoreDNSComponent, instance.Namespace, "app", names.LighthouseCoreDNSComponent, instance,
		9153, r.ScopedClient, r.RestConfig, r.Scheme, reqLogger)
	if err != nil {
		return errors.Wrap(err, "error setting up coredns metrics")
	}

	return nil
}

func (r *Reconciler) ensureLighthouseCoreDNSService(ctx context.Context, instance *submarinerv1alpha1.ServiceDiscovery,
	reqLogger logr.Logger,
) error {
	lighthouseCoreDNSService := &corev1.Service{}

	err := r.ScopedClient.Get(ctx, types.NamespacedName{Name: names.LighthouseCoreDNSComponent, Namespace: instance.Namespace},
		lighthouseCoreDNSService)
	if apierrors.IsNotFound(err) {
		lighthouseCoreDNSService = newLighthouseCoreDNSService(instance)
		if _, err = apply.Service(ctx, instance, lighthouseCoreDNSService, reqLogger,
			r.ScopedClient, r.Scheme); err != nil {
			log.Error(err, "Error creating the lighthouseCoreDNS service")

			return errors.Wrap(err, "error reconciling coredns Service")
		}
	}

	return nil
}
