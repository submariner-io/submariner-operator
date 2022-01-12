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

package nodes

import (
	"context"
	goerrors "errors"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/internal/constants"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// labels the specified worker node as a gateway node.
func LabelGateways(clientset kubernetes.Interface, gatewayNode struct{ Node string }) error {
	if gatewayNode.Node == "" {
		fmt.Printf("no gateway nodes specified, selecting one of the worker node as gateway node")

		workerNodes, err := GetAllWorkerNames(clientset)
		if err != nil {
			return err
		}

		gatewayNode.Node = workerNodes[0]
	}

	err := addLabels(clientset, gatewayNode.Node, map[string]string{constants.SubmarinerGatewayLabel: constants.TrueLabel})

	return errors.Wrap(err, fmt.Sprintf("error labeling node %q as a gateway", gatewayNode.Node))
}

// GetAllWorkerNames returns all worker nodes.
func GetAllWorkerNames(clientset kubernetes.Interface) ([]string, error) {
	workerNodes, err := clientset.CoreV1().Nodes().List(
		context.TODO(), metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/worker"})
	if err != nil {
		return nil, errors.Wrap(err, "error listing Nodes")
	}

	if len(workerNodes.Items) == 0 {
		// In some deployments (like KIND), worker nodes are not explicitly labelled. So list non-master nodes.
		workerNodes, err = clientset.CoreV1().Nodes().List(
			context.TODO(), metav1.ListOptions{LabelSelector: "!node-role.kubernetes.io/master"})
		if err != nil {
			return nil, errors.Wrap(err, "error listing Nodes")
		}
	}

	return getNodeNames(workerNodes), nil
}

func getNodeNames(nodes *corev1.NodeList) []string {
	names := []string{}
	for i := range nodes.Items {
		names = append(names, nodes.Items[i].GetName())
	}

	return names
}

// ListGateways returns the names of all node labeled as a gateway.
func ListGateways(clientset kubernetes.Interface) ([]string, error) {
	selector := labels.SelectorFromSet(map[string]string{constants.SubmarinerGatewayLabel: constants.TrueLabel})

	labeledNodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, errors.Wrap(err, "error listing Nodes")
	}

	return getNodeNames(labeledNodes), nil
}

// this function was sourced from:
// https://github.com/kubernetes/kubernetes/blob/a3ccea9d8743f2ff82e41b6c2af6dc2c41dc7b10/test/utils/density_utils.go#L36
func addLabels(clientset kubernetes.Interface, nodeName string, labelsToAdd map[string]string) error {
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
		_, lastErr = clientset.CoreV1().Nodes().Patch(context.TODO(), nodeName, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
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
