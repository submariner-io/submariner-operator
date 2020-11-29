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

package scc

import (
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/scc"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop/serviceaccount"
	"k8s.io/client-go/rest"
)

func Ensure(restConfig *rest.Config, namespace string) (bool, error) {
	updateOperatorSCC, err := scc.UpdateSCC(restConfig, namespace, serviceaccount.OperatorServiceAccount)
	if err != nil {
		return false, err
	}

	updateEngineSCC, err := scc.UpdateSCC(restConfig, namespace, serviceaccount.SubmarinerServiceAccount)
	if err != nil {
		return false, err
	}

	updateRouteAgentSCC, err := scc.UpdateSCC(restConfig, namespace, serviceaccount.RouteAgentServiceAccount)
	if err != nil {
		return false, err
	}

	updateGlobalnetSCC, err := scc.UpdateSCC(restConfig, namespace, serviceaccount.GlobalnetServiceAccount)
	if err != nil {
		return false, err
	}

	updateNPSyncerSCC, err := scc.UpdateSCC(restConfig, namespace, serviceaccount.NPSyncerServiceAccount)
	if err != nil {
		return false, err
	}

	return updateOperatorSCC || updateEngineSCC || updateRouteAgentSCC || updateGlobalnetSCC || updateNPSyncerSCC, err
}
