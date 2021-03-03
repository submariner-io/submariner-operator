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
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/scc"
	"k8s.io/client-go/rest"
)

func Ensure(restConfig *rest.Config, namespace string) (bool, error) {
	operatorSaName, err := embeddedyamls.GetObjectName(embeddedyamls.Config_rbac_submariner_operator_service_account_yaml)
	if err != nil {
		return false, err
	}

	gatewaySaName, err := embeddedyamls.GetObjectName(embeddedyamls.Config_rbac_submariner_gateway_service_account_yaml)
	if err != nil {
		return false, err
	}

	routeAgentSaName, err := embeddedyamls.GetObjectName(embeddedyamls.Config_rbac_submariner_route_agent_service_account_yaml)
	if err != nil {
		return false, err
	}

	globalnetSaName, err := embeddedyamls.GetObjectName(embeddedyamls.Config_rbac_submariner_globalnet_service_account_yaml)
	if err != nil {
		return false, err
	}

	npSyncerSaName, err := embeddedyamls.GetObjectName(embeddedyamls.Config_rbac_networkplugin_syncer_service_account_yaml)
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
