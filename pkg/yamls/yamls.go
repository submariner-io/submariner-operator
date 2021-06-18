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

import _ "embed"

//go:embed deploy/crds/submariner.io_brokers.yaml
var Deploy_crds_submariner_io_brokers_yaml string

//go:embed deploy/crds/submariner.io_submariners.yaml
var Deploy_crds_submariner_io_submariners_yaml string

//go:embed deploy/crds/submariner.io_servicediscoveries.yaml
var Deploy_crds_submariner_io_servicediscoveries_yaml string

//go:embed deploy/submariner/crds/submariner.io_clusters.yaml
var Deploy_submariner_crds_submariner_io_clusters_yaml string

//go:embed deploy/submariner/crds/submariner.io_endpoints.yaml
var Deploy_submariner_crds_submariner_io_endpoints_yaml string

//go:embed deploy/submariner/crds/submariner.io_gateways.yaml
var Deploy_submariner_crds_submariner_io_gateways_yaml string

//go:embed deploy/submariner/crds/submariner.io_clusterglobalegressips.yaml
var Deploy_submariner_crds_submariner_io_clusterglobalegressips_yaml string

//go:embed deploy/submariner/crds/submariner.io_globalegressips.yaml
var Deploy_submariner_crds_submariner_io_globalegressips_yaml string

//go:embed deploy/submariner/crds/submariner.io_globalingressips.yaml
var Deploy_submariner_crds_submariner_io_globalingressips_yaml string

//go:embed deploy/mcsapi/crds/multicluster.x_k8s.io_serviceexports.yaml
var Deploy_mcsapi_crds_multicluster_x_k8s_io_serviceexports_yaml string

//go:embed deploy/mcsapi/crds/multicluster.x_k8s.io_serviceimports.yaml
var Deploy_mcsapi_crds_multicluster_x_k8s_io_serviceimports_yaml string

//go:embed config/broker/broker-admin/service_account.yaml
var Config_broker_broker_admin_service_account_yaml string

//go:embed config/broker/broker-admin/role.yaml
var Config_broker_broker_admin_role_yaml string

//go:embed config/broker/broker-admin/role_binding.yaml
var Config_broker_broker_admin_role_binding_yaml string

//go:embed config/broker/broker-client/service_account.yaml
var Config_broker_broker_client_service_account_yaml string

//go:embed config/broker/broker-client/role.yaml
var Config_broker_broker_client_role_yaml string

//go:embed config/broker/broker-client/role_binding.yaml
var Config_broker_broker_client_role_binding_yaml string

//go:embed config/rbac/submariner-operator/service_account.yaml
var Config_rbac_submariner_operator_service_account_yaml string

//go:embed config/rbac/submariner-operator/role.yaml
var Config_rbac_submariner_operator_role_yaml string

//go:embed config/rbac/submariner-operator/role_binding.yaml
var Config_rbac_submariner_operator_role_binding_yaml string

//go:embed config/rbac/submariner-operator/cluster_role.yaml
var Config_rbac_submariner_operator_cluster_role_yaml string

//go:embed config/rbac/submariner-operator/cluster_role_binding.yaml
var Config_rbac_submariner_operator_cluster_role_binding_yaml string

//go:embed config/rbac/submariner-gateway/service_account.yaml
var Config_rbac_submariner_gateway_service_account_yaml string

//go:embed config/rbac/submariner-gateway/role.yaml
var Config_rbac_submariner_gateway_role_yaml string

//go:embed config/rbac/submariner-gateway/role_binding.yaml
var Config_rbac_submariner_gateway_role_binding_yaml string

//go:embed config/rbac/submariner-gateway/cluster_role.yaml
var Config_rbac_submariner_gateway_cluster_role_yaml string

//go:embed config/rbac/submariner-gateway/cluster_role_binding.yaml
var Config_rbac_submariner_gateway_cluster_role_binding_yaml string

//go:embed config/rbac/submariner-route-agent/service_account.yaml
var Config_rbac_submariner_route_agent_service_account_yaml string

//go:embed config/rbac/submariner-route-agent/role.yaml
var Config_rbac_submariner_route_agent_role_yaml string

//go:embed config/rbac/submariner-route-agent/role_binding.yaml
var Config_rbac_submariner_route_agent_role_binding_yaml string

//go:embed config/rbac/submariner-route-agent/cluster_role.yaml
var Config_rbac_submariner_route_agent_cluster_role_yaml string

//go:embed config/rbac/submariner-route-agent/cluster_role_binding.yaml
var Config_rbac_submariner_route_agent_cluster_role_binding_yaml string

//go:embed config/rbac/submariner-globalnet/service_account.yaml
var Config_rbac_submariner_globalnet_service_account_yaml string

//go:embed config/rbac/submariner-globalnet/role.yaml
var Config_rbac_submariner_globalnet_role_yaml string

//go:embed config/rbac/submariner-globalnet/role_binding.yaml
var Config_rbac_submariner_globalnet_role_binding_yaml string

//go:embed config/rbac/submariner-globalnet/cluster_role.yaml
var Config_rbac_submariner_globalnet_cluster_role_yaml string

//go:embed config/rbac/submariner-globalnet/cluster_role_binding.yaml
var Config_rbac_submariner_globalnet_cluster_role_binding_yaml string

//go:embed config/rbac/lighthouse-agent/service_account.yaml
var Config_rbac_lighthouse_agent_service_account_yaml string

//go:embed config/rbac/lighthouse-agent/cluster_role.yaml
var Config_rbac_lighthouse_agent_cluster_role_yaml string

//go:embed config/rbac/lighthouse-agent/cluster_role_binding.yaml
var Config_rbac_lighthouse_agent_cluster_role_binding_yaml string

//go:embed config/rbac/lighthouse-coredns/service_account.yaml
var Config_rbac_lighthouse_coredns_service_account_yaml string

//go:embed config/rbac/lighthouse-coredns/cluster_role.yaml
var Config_rbac_lighthouse_coredns_cluster_role_yaml string

//go:embed config/rbac/lighthouse-coredns/cluster_role_binding.yaml
var Config_rbac_lighthouse_coredns_cluster_role_binding_yaml string

//go:embed config/rbac/networkplugin_syncer/service_account.yaml
var Config_rbac_networkplugin_syncer_service_account_yaml string

//go:embed config/rbac/networkplugin_syncer/cluster_role.yaml
var Config_rbac_networkplugin_syncer_cluster_role_yaml string

//go:embed config/rbac/networkplugin_syncer/cluster_role_binding.yaml
var Config_rbac_networkplugin_syncer_cluster_role_binding_yaml string
