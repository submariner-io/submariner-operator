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

package embeddedyamls

import "embed"

//go:embed deploy
var embeddedFiles embed.FS

func RetrieveEmbeddedData(name string) ([]byte, error) {
	data, err := embeddedFiles.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func RetrieveEmbeddedString(name string) (string, error) {
	data, err := embeddedFiles.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

//go:embed config/rbac/submariner-operator/service_account.yaml
var ConfigRbacSubmarinerOperatorServiceAccount string

//go:embed config/rbac/submariner-operator/role.yaml
var ConfigRbacSubmarinerOperatorRole string

//go:embed config/rbac/submariner-operator/role_binding.yaml
var ConfigRbacSubmarinerOperatorRoleBinding string

//go:embed config/rbac/submariner-operator/cluster_role.yaml
var ConfigRbacSubmarinerOperatorClusterRole string

//go:embed config/rbac/submariner-operator/cluster_role_binding.yaml
var ConfigRbacSubmarinerOperatorClusterRoleBinding string

//go:embed config/rbac/submariner-gateway/service_account.yaml
var ConfigRbacSubmarinerGatewayServiceAccount string

//go:embed config/rbac/submariner-gateway/role.yaml
var ConfigRbacSubmarinerGatewayRole string

//go:embed config/rbac/submariner-gateway/role_binding.yaml
var ConfigRbacSubmarinerGatewayRoleBinding string

//go:embed config/rbac/submariner-gateway/cluster_role.yaml
var ConfigRbacSubmarinerGatewayClusterRole string

//go:embed config/rbac/submariner-gateway/cluster_role_binding.yaml
var ConfigRbacSubmarinerGatewayClusterRoleBinding string

//go:embed config/rbac/submariner-route-agent/service_account.yaml
var ConfigRbacSubmarinerRouteAgentServiceAccount string

//go:embed config/rbac/submariner-route-agent/role.yaml
var ConfigRbacSubmarinerRouteAgentRole string

//go:embed config/rbac/submariner-route-agent/role_binding.yaml
var ConfigRbacSubmarinerRouteAgentRoleBinding string

//go:embed config/rbac/submariner-route-agent/cluster_role.yaml
var ConfigRbacSubmarinerRouteAgentClusterRole string

//go:embed config/rbac/submariner-route-agent/cluster_role_binding.yaml
var ConfigRbacSubmarinerRouteAgentClusterRoleBinding string

//go:embed config/rbac/submariner-globalnet/service_account.yaml
var ConfigRbacSubmarinerGlobalnetServiceAccount string

//go:embed config/rbac/submariner-globalnet/role.yaml
var ConfigRbacSubmarinerGlobalnetRole string

//go:embed config/rbac/submariner-globalnet/role_binding.yaml
var ConfigRbacSubmarinerGlobalnetRoleBinding string

//go:embed config/rbac/submariner-globalnet/cluster_role.yaml
var ConfigRbacSubmarinerGlobalnetClusterRole string

//go:embed config/rbac/submariner-globalnet/cluster_role_binding.yaml
var ConfigRbacSubmarinerGlobalnetClusterRoleBinding string

//go:embed config/rbac/lighthouse-agent/service_account.yaml
var ConfigRbacLighthouseAgentServiceAccount string

//go:embed config/rbac/lighthouse-agent/cluster_role.yaml
var ConfigRbacLighthouseAgentClusterRole string

//go:embed config/rbac/lighthouse-agent/cluster_role_binding.yaml
var ConfigRbacLighthouseAgentClusterRoleBinding string

//go:embed config/rbac/lighthouse-coredns/service_account.yaml
var ConfigRbacLighthouseCorednsServiceAccount string

//go:embed config/rbac/lighthouse-coredns/cluster_role.yaml
var ConfigRbacLighthouseCorednsClusterRole string

//go:embed config/rbac/lighthouse-coredns/cluster_role_binding.yaml
var ConfigRbacLighthouseCorednsClusterRoleBinding string

//go:embed config/rbac/networkplugin_syncer/service_account.yaml
var ConfigRbacNetworkpluginSyncerServiceAccount string

//go:embed config/rbac/networkplugin_syncer/cluster_role.yaml
var ConfigRbacNetworkpluginSyncerClusterRole string

//go:embed config/rbac/networkplugin_syncer/cluster_role_binding.yaml
var ConfigRbacNetworkpluginSyncerClusterRoleBinding string

//go:embed config/rbac/submariner-metrics-reader/role.yaml
var ConfigRbacSubmarinerMetricsReaderRole string

//go:embed config/rbac/submariner-metrics-reader/role_binding.yaml
var Config_rbac_submariner_metrics_reader_role_binding_yaml string
