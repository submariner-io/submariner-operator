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

package kubefed

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/deployments"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/kubefedcr"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/kubefedop"
)

const deploymentCheckInterval = 5 * time.Second
const deploymentWaitTime = 10 * time.Minute

func Ensure(status *cli.Status, config *rest.Config, operatorNamespace string, operatorImage string, isController bool,
	kubeConfig string, kubeContext string) error {

	err := kubefedop.Ensure(status, config, "kubefed-operator", "quay.io/openshift/kubefed-operator:v0.1.0-rc3", isController)
	if err != nil {
		return fmt.Errorf("error deploying KubeFed operator: %s", err)
	}
	err = kubefedcr.Ensure(config, "kubefed-operator", kubeConfig, kubeContext)
	if err != nil {
		return fmt.Errorf("error deploying KubeFed CR: %s", err)
	}

	clientSet, err := clientset.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating kubefed clientset: %s", err)
	}

	if isController {
		err = deployments.WaitForReady(clientSet, "kubefed-operator", "kubefed-controller-manager", deploymentCheckInterval, deploymentWaitTime)
		if err != nil {
			return fmt.Errorf("Error deploying kubefed-controller-manager: %s", err)
		}
	}
	args := []string{"enable", "namespace"}
	if kubeConfig != "" {
		args = append(args, "--kubeconfig", kubeConfig)
	}
	if kubeContext != "" {
		args = append(args, "--host-cluster-context", kubeContext)
	}
	args = append(args, "--kubefed-namespace", "kubefed-operator")
	out, err := exec.Command("kubefedctl", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error enabling namespaces in federation: %s\n%s", err, out)
	}

	return nil
}
