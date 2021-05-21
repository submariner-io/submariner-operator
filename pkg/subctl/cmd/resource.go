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
package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	k8sV1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1opts "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/submariner-io/admiral/pkg/resource"
	submarinerv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	subClientsetv1 "github.com/submariner-io/submariner/pkg/client/clientset/versioned"

	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	subOperatorClientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
)

func getMultipleRestConfigs(kubeConfigPath string, kubeContexts []string) ([]restConfig, error) {
	var restConfigs []restConfig

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
			config, err := getClientConfigAndClusterName(rules, overrides)
			if err != nil {
				return nil, err
			}

			restConfigs = append(restConfigs, config)
		}
	}

	return restConfigs, nil
}

func getSubmarinerResourceWithError(config *rest.Config) (*v1alpha1.Submariner, error) {
	submarinerClient, err := subOperatorClientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	submariner, err := submarinerClient.SubmarinerV1alpha1().Submariners(OperatorNamespace).
		Get(context.TODO(), submarinercr.SubmarinerName, v1opts.GetOptions{})
	if err != nil {
		return nil, err
	}

	return submariner, nil
}

func getSubmarinerResource(config *rest.Config) *v1alpha1.Submariner {
	submariner, err := getSubmarinerResourceWithError(config)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		exitOnError("Error obtaining the Submariner resource", err)
	}

	return submariner
}

func getEndpointResource(config *rest.Config, clusterID string) *submarinerv1.Endpoint {
	submarinerClient, err := subClientsetv1.NewForConfig(config)
	exitOnError("Unable to get the Submariner client", err)

	endpoints, err := submarinerClient.SubmarinerV1().Endpoints(OperatorNamespace).List(context.TODO(), v1opts.ListOptions{})
	if err != nil {
		exitOnError(fmt.Sprintf("Error obtaining the Endpoints in the cluster %q", clusterID), err)
	}

	for _, endpoint := range endpoints.Items {
		if endpoint.Spec.ClusterID == clusterID {
			return &endpoint
		}
	}

	return nil
}

func getActiveGatewayNodeName(clientSet *kubernetes.Clientset, hostname string) string {
	nodes, err := clientSet.CoreV1().Nodes().List(context.TODO(), v1opts.ListOptions{})
	if err != nil {
		exitOnError("Error listing the Nodes in the local cluster", err)
	}

	for _, node := range nodes.Items {
		for _, addr := range node.Status.Addresses {
			if addr.Type == k8sV1.NodeHostName {
				if strings.HasPrefix(addr.Address, hostname) {
					return node.Name
				}
			}
		}
	}

	return ""
}

func getGatewaysResource(config *rest.Config) *submarinerv1.GatewayList {
	submarinerClient, err := subClientsetv1.NewForConfig(config)
	exitOnError("Unable to get the Submariner client", err)

	gateways, err := submarinerClient.SubmarinerV1().Gateways(OperatorNamespace).
		List(context.TODO(), v1opts.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		exitOnError("Error obtaining the Gateways resource", err)
	}

	return gateways
}

func getBrokerRestConfigAndNamespace(submariner *v1alpha1.Submariner,
	serviceDisc *v1alpha1.ServiceDiscovery) (*rest.Config, string, error) {
	if submariner != nil {
		// Try to authorize against the submariner Cluster resource as we know the CRD should exist and the credentials
		// should allow read access.
		restConfig, _, err := resource.GetAuthorizedRestConfig(submariner.Spec.BrokerK8sApiServer, submariner.Spec.BrokerK8sApiServerToken,
			submariner.Spec.BrokerK8sCA, rest.TLSClientConfig{}, schema.GroupVersionResource{
				Group:    submarinerv1.SchemeGroupVersion.Group,
				Version:  submarinerv1.SchemeGroupVersion.Version,
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

func compareFiles(file1, file2 string) (bool, error) {
	first, err := ioutil.ReadFile(file1)
	if err != nil {
		return false, err
	}
	second, err := ioutil.ReadFile(file2)
	if err != nil {
		return false, err
	}
	return bytes.Equal(first, second), nil
}
