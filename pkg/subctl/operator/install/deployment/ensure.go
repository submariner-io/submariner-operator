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

package deployment

import (
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install/embeddedyamls"
)

const deploymentCheckInterval = 5 * time.Second
const deploymentWaitTime = 2 * time.Minute

//Ensure the operator is deployed, and running
func Ensure(restConfig *rest.Config, namespace string, image string) (bool, error) {
	clientSet, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	deployment := &appsv1.Deployment{}

	err = embeddedyamls.GetObject(embeddedyamls.Operator_yaml, &deployment)

	deployment.Spec.Template.Spec.Containers[0].Image = image

	// If we are running with a local development image, don't try to pull from registry
	if image == "submariner-operator:local" {
		deployment.Spec.Template.Spec.Containers[0].ImagePullPolicy = v1.PullIfNotPresent
	}

	if err != nil {
		return false, fmt.Errorf("error parsing operator deployment yaml: %s", err)
	}

	created, err := createOrUpdateDeployment(clientSet, namespace, deployment)
	if err != nil {
		return false, err
	}

	err = waitForReadyDeployment(clientSet, namespace, deployment)

	return created, err
}

func createOrUpdateDeployment(clientSet *clientset.Clientset, namespace string, deployment *appsv1.Deployment) (bool, error) {

	_, err := clientSet.AppsV1().Deployments(namespace).Update(deployment)
	if err == nil {
		return false, nil
	} else if !errors.IsNotFound(err) {
		return false, err
	}
	_, err = clientSet.AppsV1().Deployments(namespace).Create(deployment)
	return true, err
}

func waitForReadyDeployment(clientSet *clientset.Clientset, namespace string, deployment *appsv1.Deployment) error {

	deployments := clientSet.AppsV1().Deployments(namespace)

	return wait.PollImmediate(deploymentCheckInterval, deploymentWaitTime, func() (bool, error) {
		dp, err := deployments.Get(deployment.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("error waiting for operator deployment to come up: %s", err)
		}

		for _, cond := range dp.Status.Conditions {
			if cond.Type == appsv1.DeploymentAvailable && cond.Status == v1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
}
