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
	v1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func FindPodCommandParameter(clientSet kubernetes.Interface, labelSelector, parameter string) (string, error) {
	pod, err := FindPod(clientSet, labelSelector)

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

// nolint:nilnil // Intentional as the purpose is to find.
func FindPod(clientSet kubernetes.Interface, labelSelector string) (*v1.Pod, error) {
	pods, err := clientSet.CoreV1().Pods("").List(context.TODO(), v1meta.ListOptions{
		LabelSelector: labelSelector,
		Limit:         1,
	})
	if err != nil {
		return nil, errors.WithMessagef(err, "error listing Pods by label selector %q", labelSelector)
	}

	if len(pods.Items) == 0 {
		return nil, nil
	}

	return &pods.Items[0], nil
}
