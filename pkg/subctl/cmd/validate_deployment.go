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
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var validatePodsCmd = &cobra.Command{
	Use:   "pods",
	Short: "Validate the submariner pods",
	Long:  "This command validates all the submariner pods are running",
	Run:   validatePods,
}

func init() {
	validateCmd.AddCommand(validatePodsCmd)
}

func validatePods(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContext)
	exitOnError("error getting REST config for cluster", err)

	for _, item := range configs {
		message := fmt.Sprintf("Validating Submariner Pods in %q", item.clusterName)
		status.Start(message)
		fmt.Println()
		checkPods(item.config, OperatorNamespace)
	}
}

func checkPods(config *rest.Config, operatorNamespace string) {
	submarinerResourceClient, err := submarinerclientset.NewForConfig(config)
	if err != nil {
		message := fmt.Sprintf("error creating submariner clientset: %v", err)
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return
	}

	submariner, err := submarinerResourceClient.SubmarinerV1alpha1().Submariners(operatorNamespace).
		Get("submariner", metav1.GetOptions{})

	if err != nil {
		message := fmt.Sprintf("failed to validate submariner cr due to: %v", err)
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return
	}

	kubeClientSet, err := kubernetes.NewForConfig(config)

	if err != nil {
		message := fmt.Sprintf("error creating kubernetes clientset: %v", err)
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return
	}

	CheckDaemonset(kubeClientSet, operatorNamespace, "submariner-gateway")

	CheckDaemonset(kubeClientSet, operatorNamespace, "submariner-routeagent")

	// Check if service-discovery components are deployed and running if enabled
	if submariner.Spec.ServiceDiscoveryEnabled {
		// Check lighthouse-agent
		CheckDeployment(kubeClientSet, operatorNamespace, "submariner-lighthouse-agent")

		// Check ligthouse-coreDNS
		CheckDeployment(kubeClientSet, operatorNamespace, "submariner-lighthouse-coredns")
	}
	// Check if globalnet components are deployed and running if enabled
	if submariner.Spec.GlobalCIDR != "" {
		CheckDaemonset(kubeClientSet, operatorNamespace, "submariner-globalnet")
	}
	message := "All Submariner pods are up and running"
	status.QueueFailureMessage(message)
}

func CheckDeployment(k8sClient kubernetes.Interface, namespace, deploymentName string) {
	deployment, err := k8sClient.AppsV1().Deployments(namespace).Get(deploymentName, metav1.GetOptions{})
	if err != nil {
		message := fmt.Sprintf("validation %q of deployment failed due to error: %v",
			deploymentName, err)
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return
	}

	if deployment.Status.AvailableReplicas != *deployment.Spec.Replicas {
		message := fmt.Sprintf("the configured number of replicas are not running for %q", deploymentName)
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return
	}
}

func CheckDaemonset(k8sClient kubernetes.Interface, namespace, daemonSetName string) {
	daemonSet, err := k8sClient.AppsV1().DaemonSets(namespace).Get(daemonSetName, metav1.GetOptions{})
	if err != nil {
		message := fmt.Sprintf("validation %q of daemonset failed due to error: %v", daemonSetName, err)
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return
	}

	if daemonSet.Status.CurrentNumberScheduled != daemonSet.Status.DesiredNumberScheduled {
		message := fmt.Sprintf("the desried number of daemonsets are not running for %q", daemonSetName)
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return
	}
}
