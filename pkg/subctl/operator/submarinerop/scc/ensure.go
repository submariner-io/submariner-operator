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

package scc

import (
	"github.com/submariner-io/submariner-operator/config"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/scc"
	"k8s.io/client-go/rest"
)

func Ensure(restConfig *rest.Config, namespace string) (bool, error) {
	operatorSaName, err := config.GetObjectName(config.GetEmbeddedYaml("rbac/submariner-operator/service_account.yaml"))
	if err != nil {
		return false, err
	}

	gatewaySaName, err := config.GetObjectName(config.GetEmbeddedYaml("rbac/submariner-gateway/service_account.yaml"))
	if err != nil {
		return false, err
	}

	routeAgentSaName, err := config.GetObjectName(config.GetEmbeddedYaml("rbac/submariner-route-agent/service_account.yaml"))
	if err != nil {
		return false, err
	}

	globalnetSaName, err := config.GetObjectName(config.GetEmbeddedYaml("rbac/submariner-globalnet/service_account.yaml"))
	if err != nil {
		return false, err
	}

	npSyncerSaName, err := config.GetObjectName(config.GetEmbeddedYaml("rbac/networkplugin_syncer/service_account.yaml"))
	if err != nil {
		return false, err
	}

	updateOperatorSCC, err := scc.UpdateSCC(restConfig, namespace, operatorSaName)
	if err != nil {
		return false, err
	}

	updateGatewaySCC, err := scc.UpdateSCC(restConfig, namespace, gatewaySaName)
	if err != nil {
		return false, err
	}

	updateRouteAgentSCC, err := scc.UpdateSCC(restConfig, namespace, routeAgentSaName)
	if err != nil {
		return false, err
	}

	updateGlobalnetSCC, err := scc.UpdateSCC(restConfig, namespace, globalnetSaName)
	if err != nil {
		return false, err
	}

	updateNPSyncerSCC, err := scc.UpdateSCC(restConfig, namespace, npSyncerSaName)
	if err != nil {
		return false, err
	}

	return updateOperatorSCC || updateGatewaySCC || updateRouteAgentSCC || updateGlobalnetSCC || updateNPSyncerSCC, err
}
