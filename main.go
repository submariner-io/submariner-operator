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
	"flag"
	"fmt"
	"os"
	"runtime"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/operator-framework/operator-lib/leader"
	"github.com/submariner-io/admiral/pkg/log/kzerolog"
	"github.com/submariner-io/admiral/pkg/names"
	admversion "github.com/submariner-io/admiral/pkg/version"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/metrics"
	"github.com/submariner-io/submariner-operator/controllers/servicediscovery"
	"github.com/submariner-io/submariner-operator/controllers/submariner"
	"github.com/submariner-io/submariner-operator/pkg/crd"
	"github.com/submariner-io/submariner-operator/pkg/gateway"
	"github.com/submariner-io/submariner-operator/pkg/lighthouse"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost       = "0.0.0.0"
	metricsPort int32 = 8383
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
	flag.StringVar(&pprofAddr, "pprof-bind-address", ":8082", "The address the profiling endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

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

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		// LeaderElectionID determines the name of the resource that leader election will use for holding the leader lock
		LeaderElectionID: "2a1e5b0d.submariner.io", // autogenerated
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{namespace: {}},
		},
		MapperProvider:   apiutil.NewDynamicRESTMapper,
		PprofBindAddress: pprofAddr,
	})
	if err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}

	log.Info("Setting up metrics services and monitors")

	// Setup the metrics services and service monitors
	name := os.Getenv("OPERATOR_NAME")

	// We need a new client using the manager's rest.Config because
	// the manager's caches haven't started yet and it won't allow
	// modifications until then
	metricsClient, err := client.New(cfg, client.Options{})
	if err != nil {
		log.Error(err, "Error obtaining a Kubernetes client")
	}

	if err := metrics.Setup(ctx, metricsClient, cfg, scheme,
		&metrics.ServiceInfo{
			Name:            name,
			Namespace:       namespace,
			ApplicationKey:  "name",
			ApplicationName: name,
			Port:            metricsPort,
		}, log); err != nil {
		log.Error(err, "Error setting up metrics services and monitors")
	}

	log.Info("Registering Components.")

	// Setup all Controllers
	if err = (&submariner.BrokerReconciler{
		Client: mgr.GetClient(),
		Config: mgr.GetConfig(),
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create controller", "controller", "Broker")
		os.Exit(1)
	}

	generalClient, _ := client.New(mgr.GetConfig(), client.Options{
		Scheme: scheme,
	})

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

	// Start the Cmd
	log.Info("Starting the Cmd.")

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
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
