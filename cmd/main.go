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
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"syscall"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/submariner-io/admiral/pkg/configmap"
	"github.com/submariner-io/admiral/pkg/global"
	"github.com/submariner-io/admiral/pkg/log/kzerolog"
	"github.com/submariner-io/admiral/pkg/names"
	"github.com/submariner-io/admiral/pkg/resource"
	admversion "github.com/submariner-io/admiral/pkg/version"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/controllers/metrics"
	"github.com/submariner-io/submariner-operator/internal/controllers/servicediscovery"
	"github.com/submariner-io/submariner-operator/internal/controllers/submariner"
	"github.com/submariner-io/submariner-operator/pkg/crd"
	"github.com/submariner-io/submariner-operator/pkg/gateway"
	"github.com/submariner-io/submariner-operator/pkg/lighthouse"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	scheme      = apiruntime.NewScheme()
	log         = logf.Log.WithName("cmd")
	help        = false
	version     = "devel"
	showVersion = false
)

func printVersion() {
	log.Info("Go Version: " + runtime.Version())
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

func init() {
	flag.BoolVar(&help, "help", help, "Print usage options")
	flag.BoolVar(&showVersion, "version", showVersion, "Show version")
}

//nolint:gocyclo // No further refactors necessary
func main() {
	var enableLeaderElection bool
	var probeAddr string
	var pprofAddr string
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&pprofAddr, "pprof-bind-address", "",
		"The address the profiling endpoint binds to. Disabled by default; set e.g. 127.0.0.1:8082 for local debugging.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for the controller manager to ensure there is only one active instance.")

	kzerolog.AddFlags(nil)
	flag.Parse()

	if help {
		flag.PrintDefaults()
		return
	}

	admversion.Print(names.OperatorComponent, version)

	if showVersion {
		return
	}

	kzerolog.InitK8sLogging()
	log.Info("Starting submariner-operator")

	printVersion()

	namespace, err := getWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := ctrl.GetConfig()
	if err != nil {
		log.Error(err, "unable to get kubeconfig")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()

	// Set up the CRDs we need
	crdUpdater, err := crd.UpdaterFromRestConfig(cfg)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Creating the Lighthouse CRDs")

	if _, err = lighthouse.Ensure(ctx, crdUpdater, lighthouse.DataCluster); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Creating the Gateway CRDs")

	if err := gateway.Ensure(ctx, crdUpdater); err != nil {
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
	// These are required so that we can retrieve OCP infrastructure objects using the dynamic client
	utilruntime.Must(configv1.Install(scheme))
	// +kubebuilder:scaffold:scheme

	k8sClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Error(err, "Error creating client")
		os.Exit(1)
	}

	configMap, err := configmap.Get(ctx, resource.ForConfigMap(k8sClient, namespace), names.OperatorComponent)
	if err != nil {
		log.Error(err, "Error retrieving ConfigMap")
		os.Exit(1)
	}

	global.Init(configMap)

	metricsAddr := global.Get("metrics-bind-address", "0.0.0.0:8383")

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		// LeaderElectionID determines the name of the resource that leader election will use for holding the leader lock
		LeaderElectionID: "2a1e5b0d.submariner.io", // autogenerated
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{namespace: {}},
			ByObject: map[client.Object]cache.ByObject{&v1alpha1.Broker{}: {
				Namespaces: map[string]cache.Config{cache.AllNamespaces: {}},
			}},
		},
		MapperProvider:   apiutil.NewDynamicRESTMapper,
		PprofBindAddress: pprofAddr,
	})
	if err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}

	log.Info("Setting up metrics services and monitors")

	if err = setupMetrics(ctx, metricsAddr, cfg, namespace); err != nil {
		log.Error(err, "Failed to setup metrics")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	generalClient, _ := client.New(mgr.GetConfig(), client.Options{
		Scheme: scheme,
	})

	// Setup all Controllers
	if err = (&submariner.BrokerReconciler{
		ScopedClient:  mgr.GetClient(),
		GeneralClient: generalClient,
		Config:        mgr.GetConfig(),
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create controller", "controller", "Broker")
		os.Exit(1)
	}

	if err = submariner.NewReconciler(&submariner.Config{
		ScopedClient:  mgr.GetClient(),
		GeneralClient: generalClient,
		RestConfig:    mgr.GetConfig(),
		Scheme:        mgr.GetScheme(),
		DynClient:     dynamic.NewForConfigOrDie(mgr.GetConfig()),
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create controller", "controller", "Submariner")
		os.Exit(1)
	}

	if err = (&servicediscovery.Reconciler{
		ScopedClient:  mgr.GetClient(),
		GeneralClient: generalClient,
		Scheme:        mgr.GetScheme(),
		RestConfig:    mgr.GetConfig(),
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create controller", "controller", "ServiceDiscovery")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up health check")
		os.Exit(1) // We might not want to exit here if healthchecks are not setup.
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up ready check")
		os.Exit(1) // We might not want to exit here if ready checks are not setup.
	}

	configmap.WatchAndSignalOnChange(ctx, k8sClient, namespace, syscall.SIGINT, names.OperatorComponent)

	// Start the Cmd
	log.Info("Starting the Cmd.")

	if err := mgr.Start(ctx); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}

//nolint:gocyclo // No further refactors necessary
func setupMetrics(ctx context.Context, metricsAddr string, cfg *rest.Config, namespace string) error {
	// Handle the special "0" literal that disables metrics
	if metricsAddr == "" || metricsAddr == "0" {
		log.Info("Metrics disabled - skipping Service and ServiceMonitor creation", "address", metricsAddr)
		return nil
	}

	name := os.Getenv("OPERATOR_NAME")

	// Parse the bind address to extract host and port
	host, portStr, err := net.SplitHostPort(metricsAddr)
	if err != nil {
		return fmt.Errorf("invalid metrics-bind-address format %q: %w", metricsAddr, err)
	}

	// LookupPort supports both numeric ports ("8383") and named ports ("http-metrics")
	// Use a short timeout to avoid stalling startup on slow/unavailable resolvers
	portCtx, portCancel := context.WithTimeout(ctx, 5*time.Second)
	defer portCancel()

	port, err := net.DefaultResolver.LookupPort(portCtx, "tcp", portStr)
	if err != nil {
		// Only treat timeout errors gracefully; fail on truly invalid ports
		if errors.Is(portCtx.Err(), context.DeadlineExceeded) {
			log.Info("timeout resolving port; assuming non-loopback and proceeding with Service creation",
				"port", portStr)
			// Port string must be numeric if the resolver timed out
			// This will fail with clear error if portStr is invalid
			port64, parseErr := strconv.ParseInt(portStr, 10, 32)
			if parseErr != nil {
				return fmt.Errorf("invalid port %q in metrics-bind-address: %w", portStr, parseErr)
			}

			port = int(port64)
		} else {
			return fmt.Errorf("failed to resolve port %q: %w", portStr, err)
		}
	}

	// Check if binding to loopback interface
	// Empty host (e.g., ":8383") means bind to all interfaces, not loopback
	isLoopback := false

	if host != "" {
		// LookupHost supports both IP addresses and hostnames (e.g., "localhost")
		// Use a fresh timeout context to avoid reusing an expired context from LookupPort
		hostCtx, hostCancel := context.WithTimeout(ctx, 5*time.Second)
		defer hostCancel()

		ips, err := net.DefaultResolver.LookupHost(hostCtx, host)
		if err != nil {
			// Only treat timeout/DNS errors gracefully; fail on truly invalid hosts
			if errors.Is(hostCtx.Err(), context.DeadlineExceeded) {
				log.Info("DNS resolver timeout resolving host; assuming non-loopback and proceeding with Service creation",
					"host", host)
				// Assume non-loopback on timeout
			} else {
				return fmt.Errorf("failed to resolve host %q: %w", host, err)
			}
		} else {
			// Check if any resolved IP is a loopback address
			for _, ipStr := range ips {
				if ip := net.ParseIP(ipStr); ip != nil && ip.IsLoopback() {
					isLoopback = true
					break
				}
			}
		}
	}

	if isLoopback {
		log.Info("Metrics bound to loopback - accessible only within pod via localhost or kubectl port-forward; "+
			"skipping Service and ServiceMonitor creation (no cluster scraping)", "address", metricsAddr)

		return nil
	}

	// We need a new client using the manager's rest.Config because
	// the manager's caches haven't started yet and it won't allow
	// modifications until then
	metricsClient, err := client.New(cfg, client.Options{})
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	if err := metrics.Setup(ctx, metricsClient, cfg, scheme,
		&metrics.ServiceInfo{
			Name:            name,
			Namespace:       namespace,
			ApplicationKey:  "name",
			ApplicationName: name,
			Port:            int32(port),
		}, log); err != nil {
		return fmt.Errorf("failed to setup metrics services and monitors: %w", err)
	}

	return nil
}

// getWatchNamespace returns the Namespace the operator should be watching for changes.
func getWatchNamespace() (string, error) {
	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	watchNamespaceEnvVar := "WATCH_NAMESPACE"

	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}

	return ns, nil
}
