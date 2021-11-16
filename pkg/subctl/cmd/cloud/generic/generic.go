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

package generic

import (
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/cloud-prepare/pkg/generic"
	"github.com/submariner-io/cloud-prepare/pkg/k8s"
	cloudutils "github.com/submariner-io/submariner-operator/pkg/subctl/cmd/cloud/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils/restconfig"
	"k8s.io/client-go/kubernetes"
)

func RunOnK8sCluster(kubeConfig, kubeContext string,
	function func(gwDeployer api.GatewayDeployer, reporter api.Reporter) error) error {
	k8sConfig, err := restconfig.ForCluster(kubeConfig, kubeContext)
	utils.ExitOnError("Failed to initialize a Kubernetes config", err)

	clientSet, err := kubernetes.NewForConfig(k8sConfig)
	utils.ExitOnError("Failed to create Kubernetes client", err)

	k8sClientSet := k8s.NewInterface(clientSet)

	gwDeployer := generic.NewGatewayDeployer(k8sClientSet)

	reporter := cloudutils.NewCLIReporter()

	return function(gwDeployer, reporter)
}
