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

package restconfig

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/submariner-io/admiral/pkg/resource"
	subv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"

	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
)

type RestConfig struct {
	Config      *rest.Config
	ClusterName string
}

func MustGetForClusters(kubeConfigPath string, kubeContexts []string) []RestConfig {
	configs, err := ForClusters(kubeConfigPath, kubeContexts)
	utils.ExitOnError("Error getting REST Config for cluster", err)

	return configs
}

func ForClusters(kubeConfigPath string, kubeContexts []string) ([]RestConfig, error) {
	var restConfigs []RestConfig

	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
	rules.ExplicitPath = kubeConfigPath

	contexts := []string{}
	if len(kubeContexts) > 0 {
		contexts = append(contexts, kubeContexts...)
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
			config, err := clientConfigAndClusterName(rules, overrides)
			if err != nil {
				return nil, err
			}

			restConfigs = append(restConfigs, config)
		}
	}

	return restConfigs, nil
}

func ForBroker(submariner *v1alpha1.Submariner, serviceDisc *v1alpha1.ServiceDiscovery) (*rest.Config, string, error) {
	if submariner != nil {
		// Try to authorize against the submariner Cluster resource as we know the CRD should exist and the credentials
		// should allow read access.
		restConfig, _, err := resource.GetAuthorizedRestConfig(submariner.Spec.BrokerK8sApiServer, submariner.Spec.BrokerK8sApiServerToken,
			submariner.Spec.BrokerK8sCA, rest.TLSClientConfig{}, schema.GroupVersionResource{
				Group:    subv1.SchemeGroupVersion.Group,
				Version:  subv1.SchemeGroupVersion.Version,
				Resource: "clusters",
			}, submariner.Spec.BrokerK8sRemoteNamespace)

		return restConfig, submariner.Spec.BrokerK8sRemoteNamespace, err
	}

	if serviceDisc != nil {
		// Try to authorize against the ServiceImport resource as we know the CRD should exist and the credentials
		// should allow read access.
		restConfig, _, err := resource.GetAuthorizedRestConfig(serviceDisc.Spec.BrokerK8sApiServer, serviceDisc.Spec.BrokerK8sApiServerToken,
			serviceDisc.Spec.BrokerK8sCA, rest.TLSClientConfig{}, schema.GroupVersionResource{
				Group:    "multicluster.x-k8s.io",
				Version:  "v1alpha1",
				Resource: "serviceimports",
			}, serviceDisc.Spec.BrokerK8sRemoteNamespace)

		return restConfig, serviceDisc.Spec.BrokerK8sRemoteNamespace, err
	}

	return nil, "", nil
}

func clientConfigAndClusterName(rules *clientcmd.ClientConfigLoadingRules, overrides *clientcmd.ConfigOverrides) (RestConfig, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
	clientConfig, err := config.ClientConfig()
	if err != nil {
		return RestConfig{}, err
	}

	raw, err := config.RawConfig()
	if err != nil {
		return RestConfig{}, err
	}

	clusterName := ClusterNameFromContext(raw, overrides.CurrentContext)

	if clusterName == nil {
		return RestConfig{}, fmt.Errorf("could not obtain the cluster name from kube config: %#v", raw)
	}

	return RestConfig{Config: clientConfig, ClusterName: *clusterName}, nil
}

func Clients(config *rest.Config) (dynamic.Interface, kubernetes.Interface, error) {
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}
	return dynClient, clientSet, nil
}

func ClusterNameFromContext(rawConfig api.Config, overridesContext string) *string {
	if overridesContext == "" {
		// No context provided, use the current context
		overridesContext = rawConfig.CurrentContext
	}
	configContext, ok := rawConfig.Contexts[overridesContext]
	if !ok {
		return nil
	}
	return &configContext.Cluster
}

func ForCluster(kubeConfigPath, kubeContext string) (*rest.Config, error) {
	return ClientConfig(kubeConfigPath, kubeContext).ClientConfig()
}

// ClientConfig returns a clientcmd.ClientConfig to use when communicating with K8s
func ClientConfig(kubeConfigPath, kubeContext string) clientcmd.ClientConfig {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.ExplicitPath = kubeConfigPath

	rules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
	if kubeContext != "" {
		overrides.CurrentContext = kubeContext
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
}
