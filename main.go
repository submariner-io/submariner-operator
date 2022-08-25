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

	operatorclient "github.com/openshift/cluster-dns-operator/pkg/operator/client"
	"github.com/operator-framework/operator-lib/leader"
	"github.com/submariner-io/admiral/pkg/log/kzerolog"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/metrics"
	"github.com/submariner-io/submariner-operator/controllers/servicediscovery"
	"github.com/submariner-io/submariner-operator/controllers/submariner"
	"github.com/submariner-io/submariner-operator/pkg/crd"
	"github.com/submariner-io/submariner-operator/pkg/gateway"
	"github.com/submariner-io/submariner-operator/pkg/lighthouse"
	"github.com/submariner-io/submariner-operator/pkg/version"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost       = "0.0.0.0"
	metricsPort int32 = 8383
)

var (
	scheme = apiruntime.NewScheme()
	log    = logf.Log.WithName("cmd")
	help   = false
)

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

	log.Info("Setting up metrics services and monitors")

	// Setup the metrics services and service monitors
	labels := map[string]string{"name": os.Getenv("OPERATOR_NAME")}

	// We need a new client using the manager's rest.Config because
	// the manager's caches haven't started yet and it won't allow
	// modifications until then
	metricsClient, err := client.New(cfg, client.Options{})
	if err != nil {
		log.Error(err, "Error obtaining a Kubernetes client")
	}

	if err := metrics.Setup(namespace, nil, labels, metricsPort, metricsClient, cfg, scheme, log); err != nil {
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

	generalClient, _ := operatorclient.NewClient(mgr.GetConfig())

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
		Client:        mgr.GetClient(),
		GeneralClient: generalClient,
		Scheme:        mgr.GetScheme(),
		RestConfig:    mgr.GetConfig(),
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create controller", "controller", "ServiceDiscovery")
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
