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

// nolint:revive // Blank import below for 'client/auth' is intentional to init plugins.
import (
	"context"
	goerrors "errors"
	"fmt"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/coreos/go-semver/semver"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	cmdversion "github.com/submariner-io/submariner-operator/cmd/subctl"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/version"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
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

const SubmMissingMessage = "Submariner is not installed"

func Execute() error {
	return rootCmd.Execute() // nolint:wrapcheck // No need to wrap here
}

func init() {
	rootCmd.AddCommand(cmdversion.VersionCmd)

	cloudCmd := cloud.NewCommand(&kubeConfig, &kubeContext)

	AddKubeContextFlag(cloudCmd)
	rootCmd.AddCommand(cloudCmd)
}

func AddToRootCommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}

func AddKubeConfigFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", "", "absolute path(s) to the kubeconfig file(s)")
}

// AddKubeContextFlag adds a "kubeconfig" flag and a single "kubecontext" flag that can be used once and only once.
func AddKubeContextFlag(cmd *cobra.Command) {
	AddKubeConfigFlag(cmd)
	cmd.PersistentFlags().StringVar(&kubeContext, "kubecontext", "", "kubeconfig context to use")
}

// AddKubeContextMultiFlag adds a "kubeconfig" flag and a "kubecontext" flag that can be specified multiple times (or comma separated).
func AddKubeContextMultiFlag(cmd *cobra.Command, usage string) {
	AddKubeConfigFlag(cmd)

	if usage == "" {
		usage = "comma-separated list of kubeconfig contexts to use, can be specified multiple times.\n" +
			"If none specified, all contexts referenced by the kubeconfig are used"
	}

	cmd.PersistentFlags().StringSliceVar(&kubeContexts, "kubecontexts", nil, usage)
}

const (
	OperatorNamespace = "submariner-operator"
)

func handleNodeLabels(config *rest.Config) error {
	_, clientset, err := restconfig.Clients(config)
	utils.ExitOnError("Unable to set the Kubernetes cluster connection up", err)
	// List Submariner-labeled nodes
	const submarinerGatewayLabel = "submariner.io/gateway"
	const trueLabel = "true"

	selector := labels.SelectorFromSet(map[string]string{submarinerGatewayLabel: trueLabel})

	labeledNodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return errors.Wrap(err, "error listing Nodes")
	}

	if len(labeledNodes.Items) > 0 {
		fmt.Printf("* There are %d labeled nodes in the cluster:\n", len(labeledNodes.Items))

		for i := range labeledNodes.Items {
			fmt.Printf("  - %s\n", labeledNodes.Items[i].GetName())
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
			utils.ExitOnError("Error labeling the gateway node", err)
		}
	}

	return nil
}

func askForGatewayNode(clientset kubernetes.Interface) (struct{ Node string }, error) {
	// List the worker nodes and select one
	workerNodes, err := clientset.CoreV1().Nodes().List(
		context.TODO(), metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/worker"})
	if err != nil {
		return struct{ Node string }{}, errors.Wrap(err, "error listing Nodes")
	}

	if len(workerNodes.Items) == 0 {
		// In some deployments (like KIND), worker nodes are not explicitly labelled. So list non-master nodes.
		workerNodes, err = clientset.CoreV1().Nodes().List(
			context.TODO(), metav1.ListOptions{LabelSelector: "!node-role.kubernetes.io/master"})
		if err != nil {
			return struct{ Node string }{}, errors.Wrap(err, "error listing Nodes")
		}

		if len(workerNodes.Items) == 0 {
			return struct{ Node string }{}, nil
		}
	}

	if len(workerNodes.Items) == 1 {
		return struct{ Node string }{workerNodes.Items[0].GetName()}, nil
	}

	allNodeNames := []string{}
	for i := range workerNodes.Items {
		allNodeNames = append(allNodeNames, workerNodes.Items[i].GetName())
	}

	qs := []*survey.Question{
		{
			Name: "node",
			Prompt: &survey.Select{
				Message: "Which node should be used as the gateway?",
				Options: allNodeNames,
			},
		},
	}

	answers := struct {
		Node string
	}{}

	err = survey.Ask(qs, &answers)
	if err != nil {
		return struct{ Node string }{}, err // nolint:wrapcheck // No need to wrap here
	}

	return answers, nil
}

// this function was sourced from:
// https://github.com/kubernetes/kubernetes/blob/a3ccea9d8743f2ff82e41b6c2af6dc2c41dc7b10/test/utils/density_utils.go#L36
func addLabelsToNode(c kubernetes.Interface, nodeName string, labelsToAdd map[string]string) error {
	tokens := make([]string, 0, len(labelsToAdd))
	for k, v := range labelsToAdd {
		tokens = append(tokens, fmt.Sprintf("%q:%q", k, v))
	}

	labelString := "{" + strings.Join(tokens, ",") + "}"
	patch := fmt.Sprintf(`{"metadata":{"labels":%v}}`, labelString)

	// retry is necessary because nodes get updated every 10 seconds, and a patch can happen
	// in the middle of an update

	var lastErr error
	err := wait.ExponentialBackoff(nodeLabelBackoff, func() (bool, error) {
		_, lastErr = c.CoreV1().Nodes().Patch(context.TODO(), nodeName, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
		if lastErr != nil {
			if !k8serrors.IsConflict(lastErr) {
				return false, lastErr // nolint:wrapcheck // No need to wrap here
			}
			return false, nil
		}

		return true, nil
	})

	if goerrors.Is(err, wait.ErrWaitTimeout) {
		return lastErr // nolint:wrapcheck // No need to wrap here
	}

	return err // nolint:wrapcheck // No need to wrap here
}

var nodeLabelBackoff wait.Backoff = wait.Backoff{
	Steps:    10,
	Duration: 1 * time.Second,
	Factor:   1.2,
	Jitter:   1,
}

func CheckVersionMismatch(cmd *cobra.Command, args []string) error {
	config, err := restconfig.ForCluster(kubeConfig, kubeContext)
	utils.ExitOnError("The provided kubeconfig is invalid", err)

	submariner := getSubmarinerResource(config)

	if submariner != nil && submariner.Spec.Version != "" {
		subctlVer, _ := semver.NewVersion(version.Version)
		submarinerVer, _ := semver.NewVersion(submariner.Spec.Version)

		if subctlVer != nil && submarinerVer != nil && subctlVer.LessThan(*submarinerVer) {
			return fmt.Errorf(
				"the subctl version %q is older than the deployed Submariner version %q. Please upgrade your subctl version",
				version.Version, submariner.Spec.Version)
		}
	}

	return nil
}
