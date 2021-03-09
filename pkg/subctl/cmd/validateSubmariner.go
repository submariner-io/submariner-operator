/*
© 2021 Red Hat, Inc. and others.

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
	"fmt"

	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	subMScheme "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned/scheme"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	k8sScheme "k8s.io/client-go/kubernetes/scheme"
)

func validateSubmariner(cmd *cobra.Command, args []string) {
	fmt.Println("\nValidating Submariner Configuration")
	fmt.Println("\n----------------------------")
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContext)
	exitOnError("error getting REST config for cluster", err)

	for _, item := range configs {
		localClient, err := dynamic.NewForConfig(item.config)
		if err != nil {
			exitOnError("failed: error creating dynamic client from kubeConfig: %v\n", err)
		}
		verifyDeployments(localClient, OperatorNamespace, item.clusterName)
	}
	fmt.Println("\nValidation passed.")
}

func verifyDeployments(localClient dynamic.Interface, operatorNamespace, clusterName string) {
	submarinerResourceClient := localClient.Resource(schema.GroupVersionResource{Group: "submariner.io",
		Version: "v1alpha1", Resource: "submariners"}).Namespace(operatorNamespace)
	subMObj, err := submarinerResourceClient.Get("submariner", metav1.GetOptions{})
	if err != nil {
		exitOnError("failed to validate submariner cr due to", err)
	}
	submariner := &v1alpha1.Submariner{}
	submarinerScheme := subMScheme.Scheme
	err = submarinerScheme.Convert(subMObj, submariner, nil)
	if err != nil {
		exitOnError("failed to validate submariner cr due to", err)
	}

	// Check if submariner components are deployed and running
	fmt.Printf("\nValidating Submariner Pods for cluster: %q", clusterName)
	CheckDaemonset(localClient, operatorNamespace, "submariner-gateway")

	CheckDaemonset(localClient, operatorNamespace, "submariner-routeagent")

	// Check if service-discovery components are deployed and running if enabled
	if submariner.Spec.ServiceDiscoveryEnabled {
		fmt.Printf("\nValidating Service Discovery Pods for cluster: %q", clusterName)
		// Check lighthouse-agent
		CheckDeployment(localClient, operatorNamespace, "submariner-lighthouse-agent")

		// Check ligthouse-coreDNS
		CheckDeployment(localClient, operatorNamespace, "submariner-lighthouse-coredns")
	}
	// Check if globalnet components are deployed and running if enabled
	if submariner.Spec.GlobalCIDR != "" {
		fmt.Printf("\nValidating Global Pods for cluster: %q", clusterName)
		CheckDaemonset(localClient, operatorNamespace, "submariner-globalnet")
	}
	fmt.Println()
}

func CheckDeployment(localClient dynamic.Interface, namespace, deploymentName string) {
	k8sscheme := k8sScheme.Scheme
	resourceClient := localClient.Resource(schema.GroupVersionResource{Group: "apps",
		Version: "v1", Resource: "deployments"}).Namespace(namespace)
	agentObj, err := resourceClient.Get(deploymentName, metav1.GetOptions{})
	if err != nil {
		err = fmt.Errorf("validation %q of deployment failed due to error: %v", deploymentName, err)
		exitOnError("failed", err)
	}
	deployment := &appsv1.Deployment{}
	err = k8sscheme.Convert(agentObj, deployment, nil)
	if err != nil {
		err = fmt.Errorf("validation %q of deployment failed due to error: %v", deploymentName, err)
		exitOnError("failed", err)
	}

	if deployment.Status.AvailableReplicas != *deployment.Spec.Replicas {
		err = fmt.Errorf("the configured number of replicas are not running for %q", deploymentName)
		exitOnError("failed", err)
	}
}

func CheckDaemonset(localClient dynamic.Interface, namespace, daemonSetName string) {
	k8sscheme := k8sScheme.Scheme
	resourceClient := localClient.Resource(schema.GroupVersionResource{Group: "apps",
		Version: "v1", Resource: "daemonsets"}).Namespace(namespace)
	agentObj, err := resourceClient.Get(daemonSetName, metav1.GetOptions{})
	if err != nil {
		err = fmt.Errorf("validation %q of daemonset failed due to error: %v", daemonSetName, err)
		exitOnError("failed", err)
	}
	daemonSet := &appsv1.DaemonSet{}
	err = k8sscheme.Convert(agentObj, daemonSet, nil)
	if err != nil {
		err = fmt.Errorf("validation %q of daemonset failed due to error: %v", daemonSetName, err)
		exitOnError("failed", err)
	}

	if daemonSet.Status.CurrentNumberScheduled != daemonSet.Status.DesiredNumberScheduled {
		err = fmt.Errorf("the desried number of daemonsets are not running for %q", daemonSetName)
		exitOnError("failed", err)
	}
}
