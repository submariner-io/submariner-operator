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
	"os"
	"path/filepath"
	"strings"
	"time"

	"fmt"

	"github.com/AlecAivazis/survey/v2"
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

	"github.com/spf13/cobra"
)

var (
	kubeConfig string
	rootCmd    = &cobra.Command{
		Use:   "subctl",
		Short: "An installer for Submariner",
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", kubeConfigFile(), "absolute path(s) to the kubeconfig file(s)")
}

const (
	DefaultOperatorImage = "quay.io/submariner/submariner-operator:0.0.1"
	OperatorNamespace    = "submariner-operator"
)

var operatorImage string

func kubeConfigFile() string {
	var kubeconfig string
	if kubeconfig = os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}
	if home := homeDir(); home != "" {
		return filepath.Join(home, ".kube", "config")
	} else {
		return ""
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func panicOnError(err error) {
	if err != nil {
		panic(err.Error())
	}
}

// exitOnError will print your error nicely and exit
func exitOnError(format string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, format, err.Error())
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}
}

func getClients() (dynamic.Interface, kubernetes.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return nil, nil, err
	}
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

func getRestConfig() (*rest.Config, error) {
	return clientcmd.BuildConfigFromFlags("", kubeConfig)
}

func handleNodeLabels() error {
	_, clientset, err := getClients()
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
		err = addLabelsToNode(clientset, answer.Node, map[string]string{submarinerGatewayLabel: trueLabel})
		exitOnError("Error labeling the gateway node", err)

	}
	return nil
}

func askForGatewayNode(clientset kubernetes.Interface) (struct{ Node string }, error) {
	// List all nodes and select one
	allNodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: "!node-role.kubernetes.io/master"})
	if err != nil {
		return struct{ Node string }{}, err
	}
	if len(allNodes.Items) == 0 {
		return struct{ Node string }{}, fmt.Errorf("there are no non-master nodes in the cluster")
	}
	allNodeNames := []string{}
	for _, node := range allNodes.Items {
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
func addLabelsToNode(c kubernetes.Interface, nodeName string, labels map[string]string) error {

	var tokens []string
	for k, v := range labels {
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
