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

package network

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func FindPodCommandParameter(ctx context.Context, client controllerClient.Client, labelSelector, parameter string) (string, error) {
	pod, err := FindPod(ctx, client, labelSelector)

	if err != nil || pod == nil {
		return "", err
	}

	for i := range pod.Spec.Containers {
		for _, arg := range pod.Spec.Containers[i].Command {
			if strings.HasPrefix(arg, parameter) {
				return strings.Split(arg, "=")[1], nil
			}
			// Handling the case where the command is in the form of /bin/sh -c exec ....
			if strings.Contains(arg, " ") {
				for _, subArg := range strings.Split(arg, " ") {
					if strings.HasPrefix(subArg, parameter) {
						return strings.Split(subArg, "=")[1], nil
					}
				}
			}
		}
	}

	return "", nil
}

//nolint:nilnil // Intentional as the purpose is to find.
func FindPod(ctx context.Context, client controllerClient.Client, labelSelector string) (*corev1.Pod, error) {
	selector, err := labels.Parse(labelSelector)
	if err != nil {
		return nil, errors.WithMessagef(err, "error parsing label selector %q", labelSelector)
	}

	pods := &corev1.PodList{}

	err = client.List(ctx, pods, controllerClient.InNamespace(""), controllerClient.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return nil, errors.WithMessagef(err, "error listing Pods by label selector %q", labelSelector)
	}

	if len(pods.Items) == 0 {
		return nil, nil
	}

	return &pods.Items[0], nil
}
