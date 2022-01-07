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

package deployment

import (
	"context"
	"time"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func AwaitReady(kubeClient kubernetes.Interface, namespace, deployment string, interval, timeout time.Duration) error {
	deployments := kubeClient.AppsV1().Deployments(namespace)

	// nolint:wrapcheck // No need to wrap here
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		dp, err := deployments.Get(context.TODO(), deployment, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return false, errors.Wrap(err, "error waiting for controller deployment to come up")
		}

		for _, cond := range dp.Status.Conditions {
			if cond.Type == appsv1.DeploymentAvailable && cond.Status == v1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
}
