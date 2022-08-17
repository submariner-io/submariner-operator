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

package main

import (
	"context"
	goerrors "errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/operator-framework/operator-lib/leader"
	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/log/kzerolog"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers"
	"github.com/submariner-io/submariner-operator/controllers/submariner"
	"github.com/submariner-io/submariner-operator/pkg/crd"
	"github.com/submariner-io/submariner-operator/pkg/gateway"
	"github.com/submariner-io/submariner-operator/pkg/lighthouse"
	"github.com/submariner-io/submariner-operator/pkg/metrics"
	"github.com/submariner-io/submariner-operator/pkg/version"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	ksmetric "k8s.io/kube-state-metrics/pkg/metric"
	metricsstore "k8s.io/kube-state-metrics/pkg/metrics_store"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)

var (
	scheme = apiruntime.NewScheme()
	log    = logf.Log.WithName("cmd")
	help   = false
)

const (
	metricsPath = "/metrics"
	healthzPath = "/healthz"
)

type metricHandler struct {
	stores [][]*metricsstore.MetricsStore
}

func (m *metricHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resHeader := w.Header()
	// 0.0.4 is the exposition format version of prometheus
	// https://prometheus.io/docs/instrumenting/exposition_formats/#text-based-format
	resHeader.Set("Content-Type", `text/plain; version=`+"0.0.4")

	for _, stores := range m.stores {
		for _, s := range stores {
			s.WriteAll(w)
		}
	}
}

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Submariner operator version: %v", version.Version))
}

func init() {
	flag.BoolVar(&help, "help", help, "Print usage options")
}

func main() {
	kzerolog.AddFlags(nil)
	flag.Parse()

	if help {
		flag.PrintDefaults()
		return
	}

	kzerolog.InitK8sLogging()
	log.Info("Starting submariner-operator")

	printVersion()

	namespace, err := metrics.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	ctx := context.TODO()
	// Become the leader before proceeding
	err = leader.Become(ctx, "submariner-operator-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Set up the CRDs we need
	crdUpdater, err := crd.UpdaterFromRestConfig(cfg)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Creating the Lighthouse CRDs")

	if _, err = lighthouse.Ensure(crdUpdater, lighthouse.DataCluster); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Creating the Gateway CRDs")

	if err := gateway.Ensure(crdUpdater); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Setup Scheme for all resources
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	// These are required so that we can manipulate CRDs
	utilruntime.Must(apiextensions.AddToScheme(scheme))
	// These are required so that we can retrieve Gateway objects using the dynamic client
	utilruntime.Must(submv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MapperProvider:     apiutil.NewDiscoveryRESTMapper,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
		Scheme:             scheme,
	})
	if err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup all Controllers
	if err := controllers.AddToManager(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if err = serveCRMetrics(cfg); err != nil {
		log.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: metricsPort,
		}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: operatorMetricsPort,
		}},
	}

	createServiceMonitors(ctx, cfg, servicePorts, namespace)

	if err = (&submariner.BrokerReconciler{
		Client: mgr.GetClient(),
		Config: mgr.GetConfig(),
		Log:    logf.Log.WithName("controllers").WithName("Broker"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create controller", "controller", "Broker")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	// Start the Cmd
	log.Info("Starting the Cmd.")

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}

func createServiceMonitors(ctx context.Context, cfg *rest.Config, servicePorts []v1.ServicePort, namespace string) {
	// Create Service object to expose the metrics port(s).
	service, ok, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
		return
	}

	if !ok {
		return
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}

	serviceMonitors, err := metrics.CreateServiceMonitors(cfg, namespace, services)
	if err != nil {
		log.Info("Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if goerrors.Is(err, metrics.ErrServiceMonitorNotPresent) {
			log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	} else {
		log.Info("Created service monitors", "service monitors", serviceMonitors)
	}
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config) error {
	// Below function returns filtered operator/CustomResource specific GVKs.
	// For more control override the below GVK list with your own custom logic.
	filteredGVK, err := getGVKsFromAddToScheme(v1alpha1.AddToScheme)
	if err != nil {
		return errors.Wrap(err, "error getting GVKs")
	}
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := metrics.GetWatchNamespace()
	if err != nil {
		return errors.Wrap(err, "error getting operator namespace")
	}
	// To generate metrics in other namespaces, add the values below.
	ns := []string{operatorNs}
	// Generate and serve custom resource specific metrics.
	err = generateAndServeCRMetrics(cfg, ns, filteredGVK, metricsHost, operatorMetricsPort)
	if err != nil {
		return errors.Wrap(err, "error initializing metrics")
	}

	return nil
}

// getGVKsFromAddToScheme takes in the runtime scheme and filters out all generic apimachinery meta types.
// It returns just the GVK specific to this scheme.
func getGVKsFromAddToScheme(addToSchemeFunc func(*apiruntime.Scheme) error) ([]schema.GroupVersionKind, error) {
	s := apiruntime.NewScheme()
	err := addToSchemeFunc(s)
	if err != nil {
		return nil, err
	}

	ownGVKs := []schema.GroupVersionKind{}

	schemeAllKnownTypes := s.AllKnownTypes()
	for gvk := range schemeAllKnownTypes {
		if !isKubeMetaKind(gvk.Kind) {
			ownGVKs = append(ownGVKs, gvk)
		}
	}

	return ownGVKs, nil
}

func isKubeMetaKind(kind string) bool {
	if strings.HasSuffix(kind, "List") ||
		kind == "PatchOptions" ||
		kind == "GetOptions" ||
		kind == "DeleteOptions" ||
		kind == "ExportOptions" ||
		kind == "APIVersions" ||
		kind == "APIGroupList" ||
		kind == "APIResourceList" ||
		kind == "UpdateOptions" ||
		kind == "CreateOptions" ||
		kind == "Status" ||
		kind == "WatchEvent" ||
		kind == "ListOptions" ||
		kind == "APIGroup" {
		return true
	}

	return false
}

// generateAndServeCRMetrics generates CustomResource specific metrics for each custom resource GVK in operatorGVKs.
// A list of namespaces, ns, can be passed to ServeCRMetrics to scope the generated metrics. Passing nil or
// an empty list of namespaces will result in an error.
// The function also starts serving the generated collections of the metrics on given host and port.
func generateAndServeCRMetrics(cfg *rest.Config,
	ns []string,
	operatorGVKs []schema.GroupVersionKind,
	host string, port int32,
) error {
	// We have to have at least one namespace.
	if len(ns) < 1 {
		return errors.New(
			"namespaces were empty; pass at least one namespace to generate custom resource metrics")
	}
	// Create new unstructured client.
	var allStores [][]*metricsstore.MetricsStore
	log.V(1).Info("Starting collecting operator types")
	// Loop through all the possible operator/custom resource specific types.
	for _, gvk := range operatorGVKs {
		apiVersion := gvk.GroupVersion().String()
		kind := gvk.Kind
		// Generate metric based on the kind.
		metricFamilies := generateMetricFamilies(gvk.Kind)
		log.V(1).Info("Generating metric families", "apiVersion", apiVersion, "kind", kind)
		dclient, err := newClientForGVK(cfg, apiVersion, kind)
		if err != nil {
			return err
		}

		namespaced, err := isNamespaced(gvk, cfg)
		if err != nil {
			return err
		}

		var gvkStores []*metricsstore.MetricsStore
		if namespaced {
			gvkStores = newNamespacedMetricsStores(dclient, ns, apiVersion, kind, metricFamilies)
		} else {
			gvkStores = newClusterScopedMetricsStores(dclient, metricFamilies)
		}
		// Generate collector based on the group/version, kind and the metric families.

		allStores = append(allStores, gvkStores)
	}
	// Start serving metrics.
	log.V(1).Info("Starting serving custom resource metrics")
	go serveMetrics(allStores, host, port)

	return nil
}

func serveMetrics(stores [][]*metricsstore.MetricsStore, host string, port int32) {
	listenAddress := net.JoinHostPort(host, fmt.Sprint(port))
	mux := http.NewServeMux()
	// Add metricsPath
	mux.Handle(metricsPath, &metricHandler{stores})
	// Add healthzPath
	mux.HandleFunc(healthzPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if _, err := w.Write([]byte("ok")); err != nil {
			log.Error(err, "Unable to write to serve custom metrics")
		}
	})
	// Add index
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`<html>
             <head><title>Operator SDK Metrics</title></head>
             <body>
             <h1>kube-metrics</h1>
			 <ul>
             <li><a href='` + metricsPath + `'>metrics</a></li>
             <li><a href='` + healthzPath + `'>healthz</a></li>
			 </ul>
             </body>
             </html>`))
		if err != nil {
			log.Error(err, "Unable to write to serve custom metrics")
		}
	})
	err := http.ListenAndServe(listenAddress, mux)
	log.Error(err, "Failed to serve custom metrics")
}

// newClusterScopedMetricsStores returns collections of metrics per the api/kind resource.
// The metrics are registered in the custom metrics.FamilyGenerator that needs to be defined.
func newClusterScopedMetricsStores(dclient dynamic.NamespaceableResourceInterface, metricFamily []ksmetric.FamilyGenerator,
) []*metricsstore.MetricsStore {
	var stores []*metricsstore.MetricsStore
	// Generate collector per cluster scoped resources.
	composedMetricGenFuncs := ksmetric.ComposeMetricGenFuncs(metricFamily)
	headers := ksmetric.ExtractMetricFamilyHeaders(metricFamily)
	store := metricsstore.NewMetricsStore(headers, composedMetricGenFuncs)
	reflectorPerClusterScoped(context.TODO(), dclient, &unstructured.Unstructured{}, store)
	stores = append(stores, store)

	return stores
}

func reflectorPerClusterScoped(
	ctx context.Context,
	dynamicInterface dynamic.NamespaceableResourceInterface,
	expectedType interface{},
	store cache.Store,
) {
	lw := clusterScopeListWatchFunc(dynamicInterface)
	reflector := cache.NewReflector(&lw, expectedType, store, 0)
	go reflector.Run(ctx.Done())
}

func clusterScopeListWatchFunc(dynamicInterface dynamic.NamespaceableResourceInterface) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (apiruntime.Object, error) {
			return dynamicInterface.List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return dynamicInterface.Watch(context.TODO(), opts)
		},
	}
}

// newNamespacedMetricsStores returns collections of metrics in the namespaces provided, per the api/kind resource.
// The metrics are registered in the custom metrics.FamilyGenerator that needs to be defined.
func newNamespacedMetricsStores(dclient dynamic.NamespaceableResourceInterface, namespaces []string,
	api string, kind string, metricFamily []ksmetric.FamilyGenerator,
) []*metricsstore.MetricsStore {
	namespaces = deduplicateNamespaces(namespaces)
	var stores []*metricsstore.MetricsStore
	// Generate collector per namespace.
	for _, ns := range namespaces {
		composedMetricGenFuncs := ksmetric.ComposeMetricGenFuncs(metricFamily)
		headers := ksmetric.ExtractMetricFamilyHeaders(metricFamily)
		store := metricsstore.NewMetricsStore(headers, composedMetricGenFuncs)
		reflectorPerNamespace(context.TODO(), dclient, &unstructured.Unstructured{}, store, ns)
		stores = append(stores, store)
	}

	return stores
}

func reflectorPerNamespace(
	ctx context.Context,
	dynamicInterface dynamic.NamespaceableResourceInterface,
	expectedType interface{},
	store cache.Store,
	ns string,
) {
	lw := namespacedListWatchFunc(dynamicInterface, ns)
	reflector := cache.NewReflector(&lw, expectedType, store, 0)

	go reflector.Run(ctx.Done())
}

func namespacedListWatchFunc(dynamicInterface dynamic.NamespaceableResourceInterface,
	namespace string,
) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (apiruntime.Object, error) {
			return dynamicInterface.Namespace(namespace).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return dynamicInterface.Namespace(namespace).Watch(context.TODO(), opts)
		},
	}
}

func deduplicateNamespaces(ns []string) (list []string) {
	keys := make(map[string]struct{})
	for _, entry := range ns {
		if _, ok := keys[entry]; !ok {
			keys[entry] = struct{}{}

			list = append(list, entry)
		}
	}
	return list
}

func generateMetricFamilies(kind string) []ksmetric.FamilyGenerator {
	helpText := fmt.Sprintf("Information about the %s custom resource.", kind)
	kindName := strings.ToLower(kind)
	metricName := fmt.Sprintf("%s_info", kindName)

	return []ksmetric.FamilyGenerator{
		{
			Name: metricName,
			Type: ksmetric.Gauge,
			Help: helpText,
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				crd := obj.(*unstructured.Unstructured)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       1,
							LabelKeys:   []string{"namespace", kindName},
							LabelValues: []string{crd.GetNamespace(), crd.GetName()},
						},
					},
				}
			},
		},
	}
}

func newForConfig(c *rest.Config, groupVersion string) (dynamic.Interface, error) {
	config := rest.CopyConfig(c)

	err := setConfigDefaults(groupVersion, config)
	if err != nil {
		return nil, err
	}

	return dynamic.NewForConfig(config)
}

func setConfigDefaults(groupVersion string, config *rest.Config) error {
	gv, err := schema.ParseGroupVersion(groupVersion)
	if err != nil {
		return err
	}
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	if config.GroupVersion.Group == "" && config.GroupVersion.Version == "v1" {
		config.APIPath = "/api"
	}
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: k8sscheme.Codecs}
	return nil
}

func newClientForGVK(cfg *rest.Config, apiVersion, kind string) (dynamic.NamespaceableResourceInterface, error) {
	apiResourceList, apiResource, err := getAPIResource(cfg, apiVersion, kind)
	if err != nil {
		return nil, fmt.Errorf("discovering resource information failed for %s in %s: %w", kind, apiVersion, err)
	}

	dc, err := newForConfig(cfg, apiResourceList.GroupVersion)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client failed for %s: %w", apiResourceList.GroupVersion, err)
	}

	gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
	if err != nil {
		return nil, fmt.Errorf("parsing GroupVersion %s failed: %w", apiResourceList.GroupVersion, err)
	}

	gvr := schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: apiResource.Name,
	}

	return dc.Resource(gvr), nil
}

func getAPIResource(cfg *rest.Config, apiVersion, kind string) (*metav1.APIResourceList, *metav1.APIResource, error) {
	kclient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	_, apiResourceLists, err := kclient.Discovery().ServerGroupsAndResources()
	if err != nil {
		return nil, nil, err
	}

	for _, apiResourceList := range apiResourceLists {
		if apiResourceList.GroupVersion == apiVersion {
			for _, r := range apiResourceList.APIResources {
				if r.Kind == kind {
					return apiResourceList, &r, nil
				}
			}
		}
	}

	return nil, nil, fmt.Errorf("apiVersion %s and kind %s not found available in Kubernetes cluster",
		apiVersion, kind)
}

func isNamespaced(gvk schema.GroupVersionKind, cfg *rest.Config) (bool, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		log.Error(err, "Unable to get discovery client")
		return false, err
	}
	resourceList, err := discoveryClient.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		log.Error(err, "Unable to get resource list for", "apiversion", gvk.GroupVersion().String())
		return false, err
	}
	for _, apiResource := range resourceList.APIResources {
		if apiResource.Kind == gvk.Kind {
			return apiResource.Namespaced, nil
		}
	}
	return false, errors.New("unable to find type: " + gvk.String() + " in server")
}
