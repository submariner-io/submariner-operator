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
	"os"
	"runtime"
	"syscall"

	// TODO: in operator-sdk v1 the below utilities were moved to internal.
	"github.com/operator-framework/operator-lib/leader"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/log/kzerolog"
	"github.com/submariner-io/submariner-operator/api"
	submarinerv1alpha1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers"
	"github.com/submariner-io/submariner-operator/controllers/submariner"
	"github.com/submariner-io/submariner-operator/pkg/lighthouse"
	"github.com/submariner-io/submariner-operator/pkg/metrics"
	crdutils "github.com/submariner-io/submariner-operator/pkg/utils/crds"
	"github.com/submariner-io/submariner-operator/pkg/version"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
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

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
	log.Info(fmt.Sprintf("Submariner operator version: %v", version.Version))
}

// nolint:wsl // block should not end with a whitespace.
func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(submarinerv1alpha1.AddToScheme(scheme))
	utilruntime.Must(apiextensions.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
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

	namespace, err := k8sutil.GetWatchNamespace()
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

	// Set up the CRDs we need
	crdUpdater, err := crdutils.NewFromRestConfig(cfg)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Creating the Lighthouse CRDs")

	updated, err := lighthouse.Ensure(crdUpdater, lighthouse.DataCluster)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if updated {
		log.Info("The Lighthouse CRDs were updated, restarting...")
		restartOperator()
	}

	// Setup Scheme for all resources
	if err := api.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

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
	filteredGVK, err := k8sutil.GetGVKsFromAddToScheme(api.AddToScheme)
	if err != nil {
		return errors.Wrap(err, "error getting GVKs")
	}
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return errors.Wrap(err, "error getting operator namespace")
	}
	// To generate metrics in other namespaces, add the values below.
	ns := []string{operatorNs}
	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, metricsHost, operatorMetricsPort)
	if err != nil {
		return errors.Wrap(err, "error initializing metrics")
	}

	return nil
}

func restartOperator() {
	binary, err := os.Executable()
	if err != nil {
		log.Error(err, "unable to find our executable")
		// We'll end up crashing and the orchestrator will restart us
		os.Exit(1)
	}

	if err = syscall.Exec(binary, os.Args, os.Environ()); err != nil {
		log.Error(err, "error restarting the operator")
		os.Exit(1)
	}

	// Something went wrong, rely on the orchestrator to restart us
	os.Exit(1)
}
