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
	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/scc"
	"k8s.io/client-go/dynamic"
)

func Ensure(dynClient dynamic.Interface, namespace string) (bool, error) {
	operatorSaName, err := embeddedyamls.GetObjectName(embeddedyamls.Config_rbac_submariner_operator_service_account_yaml)
	if err != nil {
		return false, errors.Wrap(err, "error parsing the operator ServiceAccount resource")
	}

	gatewaySaName, err := embeddedyamls.GetObjectName(embeddedyamls.Config_rbac_submariner_gateway_service_account_yaml)
	if err != nil {
		return false, errors.Wrap(err, "error parsing the gateway ServiceAccount resource")
	}

	routeAgentSaName, err := embeddedyamls.GetObjectName(embeddedyamls.Config_rbac_submariner_route_agent_service_account_yaml)
	if err != nil {
		return false, errors.Wrap(err, "error parsing the route agent ServiceAccount resource")
	}

	globalnetSaName, err := embeddedyamls.GetObjectName(embeddedyamls.Config_rbac_submariner_globalnet_service_account_yaml)
	if err != nil {
		return false, errors.Wrap(err, "error parsing the globalnet ServiceAccount resource")
	}

	npSyncerSaName, err := embeddedyamls.GetObjectName(embeddedyamls.Config_rbac_networkplugin_syncer_service_account_yaml)
	if err != nil {
		return false, errors.Wrap(err, "error parsing the networkplugin syncer ServiceAccount resource")
	}

	updateOperatorSCC, err := scc.UpdateSCC(dynClient, namespace, operatorSaName)
	if err != nil {
		return false, errors.Wrap(err, "error updating the SCC resource")
	}

	updateGatewaySCC, err := scc.UpdateSCC(dynClient, namespace, gatewaySaName)
	if err != nil {
		return false, errors.Wrap(err, "error updating the SCC resource")
	}

	updateRouteAgentSCC, err := scc.UpdateSCC(dynClient, namespace, routeAgentSaName)
	if err != nil {
		return false, errors.Wrap(err, "error updating the SCC resource")
	}

	updateGlobalnetSCC, err := scc.UpdateSCC(dynClient, namespace, globalnetSaName)
	if err != nil {
		return false, errors.Wrap(err, "error updating the SCC resource")
	}

	updateNPSyncerSCC, err := scc.UpdateSCC(dynClient, namespace, npSyncerSaName)

	return updateOperatorSCC || updateGatewaySCC || updateRouteAgentSCC || updateGlobalnetSCC || updateNPSyncerSCC,
		errors.Wrap(err, "error updating the SCC resource")
}
