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
package resource

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/submariner-io/shipyard/test/e2e/framework"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type PodConfig struct {
	Name       string
	ClientSet  *kubernetes.Clientset
	Scheduling framework.NetworkPodScheduling
	Networking framework.NetworkingType
	Namespace  string
	Command    string
	Timeout    uint
}

type NetworkPod struct {
	Pod       *v1.Pod
	Config    *PodConfig
	PodOutput string
}

func SchedulePodAwaitCompletion(config *PodConfig) (string, error) {
	if config.Scheduling == framework.InvalidScheduling {
		config.Scheduling = framework.GatewayNode
	}

	if config.Namespace == "" {
		config.Namespace = "default"
	}

	np := &NetworkPod{Config: config}
	if err := np.schedulePod(); err != nil {
		return "", err
	}

	defer np.DeletePod()
	if err := np.AwaitPodCompletion(); err != nil {
		return "", err
	}

	return np.PodOutput, nil
}

func SchedulePod(config *PodConfig) (*NetworkPod, error) {
	if config.Scheduling == framework.InvalidScheduling {
		config.Scheduling = framework.GatewayNode
	}

	if config.Namespace == "" {
		config.Namespace = "default"
	}

	np := &NetworkPod{Config: config}
	if err := np.schedulePod(); err != nil {
		return nil, err
	}

	return np, nil
}

func (np *NetworkPod) schedulePod() error {
	networkPod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: np.Config.Name,
			Labels: map[string]string{
				"app": np.Config.Name,
			},
		},
		Spec: v1.PodSpec{
			Affinity:      nodeAffinity(np.Config.Scheduling),
			RestartPolicy: v1.RestartPolicyNever,
			HostNetwork:   bool(np.Config.Networking),
			Containers: []v1.Container{
				{
					Name:    np.Config.Name,
					Image:   "quay.io/submariner/nettest:devel",
					Command: []string{"sh", "-c", "$(COMMAND) >/dev/termination-log 2>&1 || exit 0"},
					Env: []v1.EnvVar{
						{Name: "COMMAND", Value: np.Config.Command},
					},
				},
			},
			Tolerations: []v1.Toleration{{Operator: v1.TolerationOpExists}},
		},
	}

	pc := np.Config.ClientSet.CoreV1().Pods(np.Config.Namespace)
	var err error
	np.Pod, err = pc.Create(&networkPod)
	if err != nil {
		return err
	}

	err = np.awaitUntilPodScheduled()
	if err != nil {
		return err
	}
	return nil
}

func (np *NetworkPod) DeletePod() {
	pc := np.Config.ClientSet.CoreV1().Pods(np.Config.Namespace)
	_ = pc.Delete(np.Pod.Name, &metav1.DeleteOptions{})
}

func (np *NetworkPod) awaitUntilPodScheduled() error {
	pods := np.Config.ClientSet.CoreV1().Pods(np.Config.Namespace)

	pod, _, err := framework.AwaitResultOrError("await pod ready",
		func() (interface{}, error) {
			return pods.Get(np.Pod.Name, metav1.GetOptions{})
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

func (np *NetworkPod) AwaitPodCompletion() error {
	pods := np.Config.ClientSet.CoreV1().Pods(np.Config.Namespace)

	_, errorMsg, err := framework.AwaitResultOrError(
		fmt.Sprintf("await pod %q finished", np.Pod.Name), func() (interface{}, error) {
			return pods.Get(np.Pod.Name, metav1.GetOptions{})
		}, func(result interface{}) (bool, string, error) {
			np.Pod = result.(*v1.Pod)

			switch np.Pod.Status.Phase {
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
