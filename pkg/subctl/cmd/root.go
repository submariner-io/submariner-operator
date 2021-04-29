/*
© 2019 Red Hat, Inc. and others.

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
	"strings"
	"time"

	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	cmdversion "github.com/submariner-io/submariner-operator/pkg/subctl/cmd/version"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	goversion "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/pkg/version"
)

var (
	kubeConfig   string
	kubeContext  string
	kubeContexts []string
	rootCmd      = &cobra.Command{
		Use:   "subctl",
		Short: "An installer for Submariner",
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(cmdversion.Cmd)
	rootCmd.AddCommand(cloud.NewCommand(&kubeConfig, &kubeContext))
}

func addKubeconfigFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", "", "absolute path(s) to the kubeconfig file(s)")
	cmd.PersistentFlags().StringSliceVar(&kubeContexts, "kubecontext", nil, "kubeconfig context to use")
	if len(kubeContexts) > 0 {
		kubeContext = kubeContexts[0]
	}
}

func addKubecontextsFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringSliceVar(&kubeContexts, "kubecontexts", nil,
		"comma separated list of kubeconfig contexts to use. If none specified, all contexts referenced by kubeconfig are used")
}

const (
	OperatorNamespace = "submariner-operator"
)

func panicOnError(err error) {
	utils.PanicOnError(err)
}

// exitOnError will print your error nicely and exit in case of error
func exitOnError(message string, err error) {
	utils.ExitOnError(message, err)
}

func exitWithErrorMsg(message string) {
	utils.ExitWithErrorMsg(message)
}

func getClients(config *rest.Config) (dynamic.Interface, kubernetes.Interface, error) {
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

func getClusterNameFromContext(rawConfig clientcmdapi.Config, overridesContext string) *string {
	if overridesContext == "" {
		// No context provided, use the current context
		overridesContext = rawConfig.CurrentContext
	}
	context, ok := rawConfig.Contexts[overridesContext]
	if !ok {
		return nil
	}
	return &context.Cluster
}

func getRestConfig(kubeConfigPath, kubeContext string) (*rest.Config, error) {
	return utils.GetRestConfig(kubeConfigPath, kubeContext)
}

func getClientConfig(kubeConfigPath, kubeContext string) clientcmd.ClientConfig {
	return utils.GetClientConfig(kubeConfigPath, kubeContext)
}

func handleNodeLabels(config *rest.Config) error {
	_, clientset, err := getClients(config)
	exitOnError("Unable to set the Kubernetes cluster connection up", err)
	// List Submariner-labeled nodes
	const submarinerGatewayLabel = "submariner.io/gateway"
	const trueLabel = "true"
	selector := labels.SelectorFromSet(labels.Set(map[string]string{submarinerGatewayLabel: trueLabel}))
	labeledNodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}
	if len(labeledNodes.Items) > 0 {
		fmt.Printf("* There are %d labeled nodes in the cluster:\n", len(labeledNodes.Items))
		for _, node := range labeledNodes.Items {
			fmt.Printf("  - %s\n", node.GetName())
		}
	} else {
		answer, err := askForGatewayNode(clientset)
		if err != nil {
			return err
		}
		if answer.Node == "" {
			fmt.Printf("* No worker node found to label as the gateway\n")
		} else {
			err = addLabelsToNode(clientset, answer.Node, map[string]string{submarinerGatewayLabel: trueLabel})
			exitOnError("Error labeling the gateway node", err)
		}
	}
	return nil
}

func askForGatewayNode(clientset kubernetes.Interface) (struct{ Node string }, error) {
	// List the worker nodes and select one
	workerNodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/worker"})
	if err != nil {
		return struct{ Node string }{}, err
	}
	if len(workerNodes.Items) == 0 {
		// In some deployments (like KIND), worker nodes are not explicitly labelled. So list non-master nodes.
		workerNodes, err = clientset.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: "!node-role.kubernetes.io/master"})
		if err != nil {
			return struct{ Node string }{}, err
		}
		if len(workerNodes.Items) == 0 {
			return struct{ Node string }{}, nil
		}
	}

	if len(workerNodes.Items) == 1 {
		return struct{ Node string }{workerNodes.Items[0].GetName()}, nil
	}
	allNodeNames := []string{}
	for _, node := range workerNodes.Items {
		allNodeNames = append(allNodeNames, node.GetName())
	}
	var qs = []*survey.Question{
		{
			Name: "node",
			Prompt: &survey.Select{
				Message: "Which node should be used as the gateway?",
				Options: allNodeNames},
		},
	}
	answers := struct {
		Node string
	}{}
	err = survey.Ask(qs, &answers)
	if err != nil {
		return struct{ Node string }{}, err
	}
	return answers, nil
}

// this function was sourced from:
// https://github.com/kubernetes/kubernetes/blob/a3ccea9d8743f2ff82e41b6c2af6dc2c41dc7b10/test/utils/density_utils.go#L36
func addLabelsToNode(c kubernetes.Interface, nodeName string, labelsToAdd map[string]string) error {
	var tokens = make([]string, 0, len(labelsToAdd))
	for k, v := range labelsToAdd {
		tokens = append(tokens, fmt.Sprintf("\"%s\":\"%s\"", k, v))
	}

	labelString := "{" + strings.Join(tokens, ",") + "}"
	patch := fmt.Sprintf(`{"metadata":{"labels":%v}}`, labelString)

	// retry is necessary because nodes get updated every 10 seconds, and a patch can happen
	// in the middle of an update

	var lastErr error
	err := wait.ExponentialBackoff(nodeLabelBackoff, func() (bool, error) {
		_, lastErr = c.CoreV1().Nodes().Patch(nodeName, types.MergePatchType, []byte(patch))
		if lastErr != nil {
			if !errors.IsConflict(lastErr) {
				return false, lastErr
			}
			return false, nil
		} else {
			return true, nil
		}
	})

	if err == wait.ErrWaitTimeout {
		return lastErr
	}

	return err
}

var nodeLabelBackoff wait.Backoff = wait.Backoff{
	Steps:    10,
	Duration: 1 * time.Second,
	Factor:   1.2,
	Jitter:   1,
}

func checkVersionMismatch(cmd *cobra.Command, args []string) error {
	config, err := getRestConfig(kubeConfig, kubeContext)
	exitOnError("The provided kubeconfig is invalid", err)

	submariner := getSubmarinerResource(config)

	if submariner != nil && submariner.Spec.Version != "" {
		subctlVer, _ := goversion.NewVersion(version.Version)
		submarinerVer, _ := goversion.NewVersion(submariner.Spec.Version)

		if subctlVer != nil && submarinerVer != nil && subctlVer.LessThan(submarinerVer) {
			return fmt.Errorf(
				"the subctl version %q is older than the deployed Submariner version %q. Please upgrade your subctl version",
				version.Version, submariner.Spec.Version)
		}
	}

	return nil
}
