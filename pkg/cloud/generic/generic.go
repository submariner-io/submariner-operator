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
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/cloud-prepare/pkg/generic"
	"github.com/submariner-io/cloud-prepare/pkg/k8s"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"k8s.io/client-go/kubernetes"
)

func RunOnCluster(restConfigProducer *restconfig.Producer, status reporter.Interface,
	function func(api.GatewayDeployer, reporter.Interface) error,
) error {
	k8sConfig, err := restConfigProducer.ForCluster()
	if err != nil {
		return status.Error(err, "error initializing Kubernetes config")
	}

	clientSet, err := kubernetes.NewForConfig(k8sConfig.Config)
	if err != nil {
		return status.Error(err, "error creating Kubernetes client")
	}

	k8sClientSet := k8s.NewInterface(clientSet)

	gwDeployer := generic.NewGatewayDeployer(k8sClientSet)

	return function(gwDeployer, status)
}
