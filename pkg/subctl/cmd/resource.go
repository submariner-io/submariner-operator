/*
© 2021 Red Hat, Inc. and others

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
package cmd

import (
	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/pkg/names"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1opts "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	subOperatorClientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	submarinerv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	subClientsetv1 "github.com/submariner-io/submariner/pkg/client/clientset/versioned"
)

func getMultipleRestConfigs(kubeConfigPath, kubeContext string) ([]restConfig, error) {
	var restConfigs []restConfig

	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
	rules.ExplicitPath = kubeConfigPath

	contexts := []string{}
	if kubeContext != "" {
		contexts = append(contexts, kubeContext)
	} else {
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
		rawConfig, err := kubeConfig.RawConfig()
		if err != nil {
			return restConfigs, err
		}
		for context := range rawConfig.Contexts {
			contexts = append(contexts, context)
		}
	}

	for _, context := range contexts {
		if context != "" {
			overrides.CurrentContext = context
			config, err := getClientConfigAndClusterName(rules, overrides)
			if err != nil {
				return nil, err
			}

			restConfigs = append(restConfigs, config)
		}
	}

	return restConfigs, nil
}

func getSubmarinerResource(config *rest.Config) *v1alpha1.Submariner {
	submarinerClient, err := subOperatorClientset.NewForConfig(config)
	exitOnError("Unable to get the Submariner client", err)

	submariner, err := submarinerClient.SubmarinerV1alpha1().Submariners(OperatorNamespace).
		Get(submarinercr.SubmarinerName, v1opts.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		exitOnError("Error obtaining the Submariner resource", err)
	}

	return submariner
}

func getGatewaysResource(config *rest.Config) *submarinerv1.GatewayList {
	submarinerClient, err := subClientsetv1.NewForConfig(config)
	exitOnError("Unable to get the Submariner client", err)

	gateways, err := submarinerClient.SubmarinerV1().Gateways(OperatorNamespace).
		List(v1opts.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		exitOnError("Error obtaining the Gateways resource", err)
	}

	return gateways
}

func getBrokerRestConfig(localConfig *rest.Config) (*rest.Config, error) {
	submarinerClient, err := subOperatorClientset.NewForConfig(localConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "error getting submariner client")
	}

	brokerConfig, err := getBrokerRestConfigFromSubmariner(submarinerClient)
	if apierrors.IsNotFound(err) {
		return getBrokerRestConfigFromServiceDisc(submarinerClient)
	}

	return brokerConfig, err
}

func getBrokerRestConfigFromSubmariner(submarinerClient *subOperatorClientset.Clientset) (*rest.Config, error) {
	submariner, err := submarinerClient.SubmarinerV1alpha1().Submariners(OperatorNamespace).
		Get(submarinercr.SubmarinerName, v1opts.GetOptions{})
	if err != nil {
		return nil, errors.WithMessage(err, "error obtaining the Submariner resource")
	}

	// Try to authorize against the submariner Cluster resource as we know the CRD should exist and the credentials
	// should allow read access.
	restConfig, _, err := resource.GetAuthorizedRestConfig(submariner.Spec.BrokerK8sApiServer, submariner.Spec.BrokerK8sApiServerToken,
		submariner.Spec.BrokerK8sCA, rest.TLSClientConfig{}, schema.GroupVersionResource{
			Group:    submarinerv1.SchemeGroupVersion.Group,
			Version:  submarinerv1.SchemeGroupVersion.Version,
			Resource: "clusters",
		})

	return restConfig, err
}

func getBrokerRestConfigFromServiceDisc(submarinerClient *subOperatorClientset.Clientset) (*rest.Config, error) {
	serviceDisc, err := submarinerClient.SubmarinerV1alpha1().ServiceDiscoveries(OperatorNamespace).
		Get(names.ServiceDiscoveryCrName, v1opts.GetOptions{})
	if err != nil {
		return nil, errors.WithMessage(err, "error obtaining the ServiceDiscovery resource")
	}

	// Try to authorize against the ServiceImport resource as we know the CRD should exist and the credentials
	// should allow read access.
	restConfig, _, err := resource.GetAuthorizedRestConfig(serviceDisc.Spec.BrokerK8sApiServer, serviceDisc.Spec.BrokerK8sApiServerToken,
		serviceDisc.Spec.BrokerK8sCA, rest.TLSClientConfig{}, schema.GroupVersionResource{
			Group:    "multicluster.x-k8s.io",
			Version:  "v1alpha1",
			Resource: "serviceimports",
		})

	return restConfig, err
}
