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

package deployment

import (
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/deployments"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/utils"
)

const deploymentCheckInterval = 5 * time.Second
const deploymentWaitTime = 10 * time.Minute

//Ensure the lighthouse controller is deployed, and running
func Ensure(restConfig *rest.Config, namespace string, image string) (bool, error) {
	clientSet, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	deployment := &appsv1.Deployment{}

	err = embeddedyamls.GetObject(embeddedyamls.Lighthouse_controller_yaml, &deployment)

	deployment.Spec.Template.Spec.Containers[0].Image = image

	// If we are running with a local development image, don't try to pull from registry
	if getVersion(image) == "local" {
		deployment.Spec.Template.Spec.Containers[0].ImagePullPolicy = v1.PullIfNotPresent
	}

	if err != nil {
		return false, fmt.Errorf("error parsing controller deployment yaml: %s", err)
	}

	created, err := utils.CreateOrUpdateDeployment(clientSet, namespace, deployment)
	if err != nil {
		return false, err
	}

	err = deployments.WaitForReady(clientSet, namespace, deployment.Name, deploymentCheckInterval, deploymentWaitTime)

	return created, err
}

func getVersion(image string) string {
	s := strings.Split(image, ":")
	return s[len(s)-1]
}
