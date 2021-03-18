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
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var validatePodsCmd = &cobra.Command{
	Use:   "pods",
	Short: "Validate the submariner pods",
	Long:  "This command checks that all the submariner pods are running",
	Run:   validatePods,
}

func init() {
	validateCmd.AddCommand(validatePodsCmd)
}

func validatePods(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContext)
	exitOnError("error getting REST config for cluster", err)

	for _, item := range configs {
		message := fmt.Sprintf("Validating submariner pods in %q", item.clusterName)
		status.Start(message)
		fmt.Println()
		checkPods(item.config, OperatorNamespace)
	}
}

func checkPods(config *rest.Config, operatorNamespace string) {
	submariner := getSubmarinerResource(config)
	if submariner == nil {
		status.QueueWarningMessage(submMissingMessage)
		status.End(cli.Success)
		return
	}

	kubeClientSet, err := kubernetes.NewForConfig(config)

	if err != nil {
		exitOnError("error creating Kubernetes client", err)
	}

	if !CheckDaemonset(kubeClientSet, operatorNamespace, "submariner-gateway") {
		return
	}

	if !CheckDaemonset(kubeClientSet, operatorNamespace, "submariner-routeagent") {
		return
	}

	// Check if service-discovery components are deployed and running if enabled
	if submariner.Spec.ServiceDiscoveryEnabled {
		// Check lighthouse-agent
		if !CheckDeployment(kubeClientSet, operatorNamespace, "submariner-lighthouse-agent") {
			return
		}

		// Check lighthouse-coreDNS
		if !CheckDeployment(kubeClientSet, operatorNamespace, "submariner-lighthouse-coredns") {
			return
		}
	}
	// Check if globalnet components are deployed and running if enabled
	if submariner.Spec.GlobalCIDR != "" {
		if !CheckDaemonset(kubeClientSet, operatorNamespace, "submariner-globalnet") {
			return
		}
	}

	message := "All Submariner pods are up and running"
	status.QueueSuccessMessage(message)
	status.End(cli.Success)
}

func CheckDeployment(k8sClient kubernetes.Interface, namespace, deploymentName string) bool {
	deployment, err := k8sClient.AppsV1().Deployments(namespace).Get(deploymentName, metav1.GetOptions{})
	if err != nil {
		message := fmt.Sprintf("Error obtaining Deployment %q: %v", deploymentName, err)
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return false
	}

	var replicas int32 = 1
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	if deployment.Status.AvailableReplicas != replicas {
		message := fmt.Sprintf("The desired number of replicas for Deployment %q (%d)"+
			" does not match the actual number running (%d)", deploymentName, replicas,
			deployment.Status.AvailableReplicas)
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return false
	}

	return true
}

func CheckDaemonset(k8sClient kubernetes.Interface, namespace, daemonSetName string) bool {
	daemonSet, err := k8sClient.AppsV1().DaemonSets(namespace).Get(daemonSetName, metav1.GetOptions{})
	if err != nil {
		message := fmt.Sprintf("Error obtaining Daemonset %q: %v", daemonSetName, err)
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return false
	}

	if daemonSet.Status.CurrentNumberScheduled != daemonSet.Status.DesiredNumberScheduled {
		message := fmt.Sprintf("The desired number of running pods for DaemonSet %q (%d)"+
			" does not match the actual number (%d)", daemonSetName, daemonSet.Status.DesiredNumberScheduled,
			daemonSet.Status.CurrentNumberScheduled)
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return false
	}

	return true
}
