/*
© 2020 Red Hat, Inc. and others.

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

package deployments

import (
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
)

func WaitForReady(clientSet *clientset.Clientset, namespace string, deployment string, interval, timeout time.Duration) error {

	deployments := clientSet.AppsV1().Deployments(namespace)

	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		dp, err := deployments.Get(deployment, metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return false, fmt.Errorf("error waiting for controller deployment to come up: %s", err)
		}

		for _, cond := range dp.Status.Conditions {
			if cond.Type == appsv1.DeploymentAvailable && cond.Status == v1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
}
