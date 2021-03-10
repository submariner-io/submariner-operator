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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/spf13/cobra"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func validateSubmariner(cmd *cobra.Command, args []string) {
	fmt.Println("\nValidating Submariner Configuration")
	fmt.Println("\n----------------------------")
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContext)
	exitOnError("error getting REST config for cluster", err)

	for _, item := range configs {
		verifyDeployments(item.config, OperatorNamespace, item.clusterName)
	}
	fmt.Println("\nValidation passed.")
}

func verifyDeployments(config *rest.Config, operatorNamespace, clusterName string) {
	submarinerResourceClient, err := submarinerclientset.NewForConfig(config)
	if err != nil {
		exitOnError("failed: error creating submariner clientset: %v\n", err)
	}

	submariner, err := submarinerResourceClient.SubmarinerV1alpha1().Submariners(operatorNamespace).
		Get("submariner", metav1.GetOptions{})
	if err != nil {
		exitOnError("failed to validate submariner cr due to", err)
	}

	kubeClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		exitOnError("failed: error creating kubernetes clientset: %v\n", err)
	}

	// Check if submariner components are deployed and running
	fmt.Printf("\nValidating Submariner Pods for cluster: %q", clusterName)
	CheckDaemonset(kubeClientSet, operatorNamespace, "submariner-gateway")

	CheckDaemonset(kubeClientSet, operatorNamespace, "submariner-routeagent")

	// Check if service-discovery components are deployed and running if enabled
	if submariner.Spec.ServiceDiscoveryEnabled {
		fmt.Printf("\nValidating Service Discovery Pods for cluster: %q", clusterName)
		// Check lighthouse-agent
		CheckDeployment(kubeClientSet, operatorNamespace, "submariner-lighthouse-agent")

		// Check ligthouse-coreDNS
		CheckDeployment(kubeClientSet, operatorNamespace, "submariner-lighthouse-coredns")
	}
	// Check if globalnet components are deployed and running if enabled
	if submariner.Spec.GlobalCIDR != "" {
		fmt.Printf("\nValidating Globalnet Pods for cluster: %q", clusterName)
		CheckDaemonset(kubeClientSet, operatorNamespace, "submariner-globalnet")
	}
	fmt.Println()
}

func CheckDeployment(k8sClient kubernetes.Interface, namespace, deploymentName string) {
	deployment, err := k8sClient.AppsV1().Deployments(namespace).Get(deploymentName, metav1.GetOptions{})
	if err != nil {
		err = fmt.Errorf("validation %q of deployment failed due to error: %v", deploymentName, err)
		exitOnError("failed", err)
	}

	if deployment.Status.AvailableReplicas != *deployment.Spec.Replicas {
		err = fmt.Errorf("the configured number of replicas are not running for %q", deploymentName)
		exitOnError("failed", err)
	}
}

func CheckDaemonset(k8sClient kubernetes.Interface, namespace, daemonSetName string) {
	daemonSet, err := k8sClient.AppsV1().DaemonSets(namespace).Get(daemonSetName, metav1.GetOptions{})
	if err != nil {
		err = fmt.Errorf("validation %q of daemonset failed due to error: %v", daemonSetName, err)
		exitOnError("failed", err)
	}

	if daemonSet.Status.CurrentNumberScheduled != daemonSet.Status.DesiredNumberScheduled {
		err = fmt.Errorf("the desried number of daemonsets are not running for %q", daemonSetName)
		exitOnError("failed", err)
	}
}
