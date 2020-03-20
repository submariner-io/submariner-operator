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

package operatorpod

import (
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/deployments"
	"github.com/submariner-io/submariner-operator/pkg/utils"
)

const deploymentCheckInterval = 5 * time.Second
const deploymentWaitTime = 10 * time.Minute

//Ensure the operator is deployed, and running
func Ensure(restConfig *rest.Config, namespace string, operatorName string, image string) (bool, error) {
	clientSet, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	replicas := int32(1)
	imagePullPolicy := v1.PullAlways
	// If we are running with a local development image, don't try to pull from registry
	if strings.HasSuffix(image, ":local") {
		imagePullPolicy = v1.PullIfNotPresent
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      operatorName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"name": operatorName}},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"name": operatorName},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: operatorName,
					Containers: []v1.Container{
						{
							Name:            operatorName,
							Image:           image,
							Command:         []string{operatorName},
							ImagePullPolicy: imagePullPolicy,
							Env: []v1.EnvVar{
								{
									Name: "WATCH_NAMESPACE", ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								}, {
									Name: "POD_NAME", ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								}, {
									Name: "OPERATOR_NAME", Value: operatorName,
								},
							},
						},
					},
				},
			},
		},
	}

	created, err := utils.CreateOrUpdateDeployment(clientSet, namespace, deployment)
	if err != nil {
		return false, err
	}

	err = deployments.WaitForReady(clientSet, namespace, deployment.Name, deploymentCheckInterval, deploymentWaitTime)

	return created, err
}
