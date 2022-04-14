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

package pods

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/submariner-io/shipyard/test/e2e/framework"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type schedulingType int

const (
	InvalidScheduling schedulingType = iota
	GatewayNode
	NonGatewayNode
	CustomNode
)

type networkingType bool

const (
	HostNetworking networkingType = true
	PodNetworking  networkingType = false
)

type Scheduling struct {
	ScheduleOn schedulingType
	NodeName   string
	Networking networkingType
}

type Config struct {
	Name       string
	ClientSet  kubernetes.Interface
	Scheduling Scheduling
	Namespace  string
	Command    string
	Timeout    uint
}

type Scheduled struct {
	Pod       *v1.Pod
	Config    *Config
	PodOutput string
}

func ScheduleAndAwaitCompletion(config *Config) (string, error) {
	if config.Scheduling.ScheduleOn == InvalidScheduling {
		config.Scheduling.ScheduleOn = GatewayNode
	}

	if config.Namespace == "" {
		config.Namespace = "default"
	}

	np := &Scheduled{Config: config}
	if err := np.schedule(); err != nil {
		return "", err
	}

	defer np.Delete()

	if err := np.AwaitCompletion(); err != nil {
		return "", err
	}

	return np.PodOutput, nil
}

func Schedule(config *Config) (*Scheduled, error) {
	if config.Scheduling.ScheduleOn == InvalidScheduling {
		config.Scheduling.ScheduleOn = GatewayNode
	}

	if config.Namespace == "" {
		config.Namespace = "default"
	}

	np := &Scheduled{Config: config}
	if err := np.schedule(); err != nil {
		return nil, err
	}

	return np, nil
}

func (np *Scheduled) schedule() error {
	if np.Config.Scheduling.ScheduleOn == CustomNode && np.Config.Scheduling.NodeName == "" {
		return fmt.Errorf("CustomNode is specified for scheduling, but nodeName is missing")
	}

	networkPod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: np.Config.Name,
			Labels: map[string]string{
				"app": np.Config.Name,
			},
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyNever,
			HostNetwork:   bool(np.Config.Scheduling.Networking),
			Containers: []v1.Container{
				{
					Name:    np.Config.Name,
					Image:   "quay.io/submariner/nettest:feature-multi-active-gw",
					Command: []string{"sh", "-c", "$(COMMAND) >/dev/termination-log 2>&1 || exit 0"},
					Env: []v1.EnvVar{
						{Name: "COMMAND", Value: np.Config.Command},
					},
				},
			},
			Tolerations: []v1.Toleration{{Operator: v1.TolerationOpExists}},
		},
	}

	if np.Config.Scheduling.Networking == HostNetworking {
		networkPod.Spec.Containers[0].SecurityContext = &v1.SecurityContext{
			Capabilities: &v1.Capabilities{
				Add:  []v1.Capability{"NET_ADMIN", "NET_RAW"},
				Drop: []v1.Capability{"all"},
			},
		}
	}

	if np.Config.Scheduling.ScheduleOn == CustomNode {
		networkPod.Spec.NodeName = np.Config.Scheduling.NodeName
	} else {
		networkPod.Spec.Affinity = nodeAffinity(np.Config.Scheduling.ScheduleOn)
	}

	pc := np.Config.ClientSet.CoreV1().Pods(np.Config.Namespace)

	var err error

	np.Pod, err = pc.Create(context.TODO(), &networkPod, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "error creating Pod")
	}

	err = np.awaitUntilScheduled()
	if err != nil {
		return err
	}

	return nil
}

func (np *Scheduled) Delete() {
	pc := np.Config.ClientSet.CoreV1().Pods(np.Config.Namespace)
	_ = pc.Delete(context.TODO(), np.Pod.Name, metav1.DeleteOptions{})
}

// nolint:wrapcheck // No need to wrap errors here.
func (np *Scheduled) awaitUntilScheduled() error {
	pods := np.Config.ClientSet.CoreV1().Pods(np.Config.Namespace)

	pod, _, err := framework.AwaitResultOrError("await pod ready",
		func() (interface{}, error) {
			return pods.Get(context.TODO(), np.Pod.Name, metav1.GetOptions{})
		}, func(result interface{}) (bool, string, error) {
			pod := result.(*v1.Pod)
			if pod.Status.Phase != v1.PodRunning && pod.Status.Phase != v1.PodSucceeded {
				if pod.Status.Phase != v1.PodPending {
					return false, "", fmt.Errorf("unexpected pod phase %v - expected %v or %v",
						pod.Status.Phase, v1.PodPending, v1.PodRunning)
				}
				return false, fmt.Sprintf("Pod %q is still pending", pod.Name), nil
			}

			return true, "", nil // pod is either running or has completed its execution
		})
	if err != nil {
		return err
	}

	np.Pod = pod.(*v1.Pod)

	return nil
}

// nolint:wrapcheck // No need to wrap errors here.
func (np *Scheduled) AwaitCompletion() error {
	pods := np.Config.ClientSet.CoreV1().Pods(np.Config.Namespace)

	_, errorMsg, err := framework.AwaitResultOrError(
		fmt.Sprintf("await pod %q finished", np.Pod.Name), func() (interface{}, error) {
			return pods.Get(context.TODO(), np.Pod.Name, metav1.GetOptions{})
		}, func(result interface{}) (bool, string, error) {
			np.Pod = result.(*v1.Pod)

			switch np.Pod.Status.Phase { // nolint:exhaustive // 'missing cases in switch' - OK
			case v1.PodSucceeded:
				return true, "", nil
			case v1.PodFailed:
				return true, "", nil
			default:
				return false, fmt.Sprintf("Pod status is %v", np.Pod.Status.Phase), nil
			}
		})
	if err != nil {
		return errors.Wrapf(err, errorMsg)
	}

	finished := np.Pod.Status.Phase == v1.PodSucceeded || np.Pod.Status.Phase == v1.PodFailed
	if finished {
		np.PodOutput = np.Pod.Status.ContainerStatuses[0].State.Terminated.Message
	}

	return nil
}

func nodeAffinity(scheduling schedulingType) *v1.Affinity {
	var nodeSelTerms []v1.NodeSelectorTerm

	switch scheduling {
	case GatewayNode:
		nodeSelTerms = addNodeSelectorTerm(nodeSelTerms, framework.GatewayLabel,
			v1.NodeSelectorOpIn, []string{"true"})

	case NonGatewayNode:
		nodeSelTerms = addNodeSelectorTerm(nodeSelTerms, framework.GatewayLabel,
			v1.NodeSelectorOpDoesNotExist, nil)
		nodeSelTerms = addNodeSelectorTerm(nodeSelTerms, framework.GatewayLabel,
			v1.NodeSelectorOpNotIn, []string{"true"})
	case InvalidScheduling:
	case CustomNode:
	}

	return &v1.Affinity{
		NodeAffinity: &v1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
				NodeSelectorTerms: nodeSelTerms,
			},
		},
	}
}

func addNodeSelectorTerm(nodeSelTerms []v1.NodeSelectorTerm, label string,
	op v1.NodeSelectorOperator, values []string,
) []v1.NodeSelectorTerm {
	return append(nodeSelTerms, v1.NodeSelectorTerm{MatchExpressions: []v1.NodeSelectorRequirement{
		{
			Key:      label,
			Operator: op,
			Values:   values,
		},
	}})
}
