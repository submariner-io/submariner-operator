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

package serviceaccount

import (
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/serviceaccount"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	embeddedyamls "github.com/submariner-io/submariner-operator/pkg/yamls"
)

// Ensure functions updates or installs the operator CRDs in the cluster
func Ensure(restConfig *rest.Config, namespace string) (bool, error) {
	clientSet, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	createdSA, err := ensureServiceAccounts(clientSet, namespace)
	if err != nil {
		return false, err
	}

	createdRole, err := ensureRoles(clientSet, namespace)
	if err != nil {
		return false, err
	}

	createdRB, err := ensureRoleBindings(clientSet, namespace)
	if err != nil {
		return false, err
	}

	createdCR, err := ensureClusterRoles(clientSet)
	if err != nil {
		return false, err
	}

	createdCRB, err := ensureClusterRoleBindings(clientSet, namespace)
	if err != nil {
		return false, err
	}

	return createdSA || createdRole || createdRB || createdCR || createdCRB, nil
}

func ensureServiceAccounts(clientSet *clientset.Clientset, namespace string) (bool, error) {
	createdOperatorSA, err := serviceaccount.Ensure(clientSet, namespace,
		embeddedyamls.ConfigRbacSubmarinerOperatorServiceAccount)
	if err != nil {
		return false, err
	}

	createdSubmarinerSA, err := serviceaccount.Ensure(clientSet, namespace,
		embeddedyamls.ConfigRbacSubmarinerGatewayServiceAccount)
	if err != nil {
		return false, err
	}

	createdRouteAgentSA, err := serviceaccount.Ensure(clientSet, namespace,
		embeddedyamls.ConfigRbacSubmarinerRouteAgentServiceAccount)
	if err != nil {
		return false, err
	}

	createdGlobalnetSA, err := serviceaccount.Ensure(clientSet, namespace,
		embeddedyamls.ConfigRbacSubmarinerGlobalnetServiceAccount)
	if err != nil {
		return false, err
	}

	createdNPSyncerSA, err := serviceaccount.Ensure(clientSet, namespace,
		embeddedyamls.ConfigRbacNetworkpluginSyncerServiceAccount)
	return createdOperatorSA || createdSubmarinerSA || createdRouteAgentSA || createdGlobalnetSA || createdNPSyncerSA, err
}

func ensureClusterRoles(clientSet *clientset.Clientset) (bool, error) {
	createdOperatorCR, err := serviceaccount.EnsureClusterRole(clientSet,
		embeddedyamls.ConfigRbacSubmarinerOperatorClusterRole)
	if err != nil {
		return false, err
	}

	createdSubmarinerCR, err := serviceaccount.EnsureClusterRole(clientSet,
		embeddedyamls.ConfigRbacSubmarinerGatewayClusterRole)
	if err != nil {
		return false, err
	}

	createdRouteAgentCR, err := serviceaccount.EnsureClusterRole(clientSet,
		embeddedyamls.ConfigRbacSubmarinerRouteAgentClusterRole)
	if err != nil {
		return false, err
	}

	createdGlobalnetCR, err := serviceaccount.EnsureClusterRole(clientSet,
		embeddedyamls.ConfigRbacSubmarinerGlobalnetClusterRole)
	if err != nil {
		return false, err
	}

	createdNPSyncerCR, err := serviceaccount.EnsureClusterRole(clientSet,
		embeddedyamls.ConfigRbacNetworkpluginSyncerClusterRole)
	return createdOperatorCR || createdSubmarinerCR || createdRouteAgentCR || createdGlobalnetCR || createdNPSyncerCR, err
}

func ensureClusterRoleBindings(clientSet *clientset.Clientset, namespace string) (bool, error) {
	createdOperatorCRB, err := serviceaccount.EnsureClusterRoleBinding(clientSet, namespace,
		embeddedyamls.ConfigRbacSubmarinerOperatorClusterRoleBinding)
	if err != nil {
		return false, err
	}

	createdSubmarinerCRB, err := serviceaccount.EnsureClusterRoleBinding(clientSet, namespace,
		embeddedyamls.ConfigRbacSubmarinerGatewayClusterRoleBinding)
	if err != nil {
		return false, err
	}

	createdRouteAgentCRB, err := serviceaccount.EnsureClusterRoleBinding(clientSet, namespace,
		embeddedyamls.ConfigRbacSubmarinerRouteAgentClusterRoleBinding)
	if err != nil {
		return false, err
	}

	createdGlobalnetCRB, err := serviceaccount.EnsureClusterRoleBinding(clientSet, namespace,
		embeddedyamls.ConfigRbacSubmarinerGlobalnetClusterRoleBinding)
	if err != nil {
		return false, err
	}

	createdNPSyncerCRB, err := serviceaccount.EnsureClusterRoleBinding(clientSet, namespace,
		embeddedyamls.ConfigRbacNetworkpluginSyncerClusterRoleBinding)
	return createdOperatorCRB || createdSubmarinerCRB || createdRouteAgentCRB || createdGlobalnetCRB || createdNPSyncerCRB, err
}

func ensureRoles(clientSet *clientset.Clientset, namespace string) (bool, error) {
	createdOperatorRole, err := serviceaccount.EnsureRole(clientSet, namespace,
		embeddedyamls.ConfigRbacSubmarinerOperatorRole)
	if err != nil {
		return false, err
	}

	createdSubmarinerRole, err := serviceaccount.EnsureRole(clientSet, namespace,
		embeddedyamls.ConfigRbacSubmarinerGatewayRole)
	if err != nil {
		return false, err
	}

	createdRouteAgentRole, err := serviceaccount.EnsureRole(clientSet, namespace,
		embeddedyamls.ConfigRbacSubmarinerRouteAgentRole)
	if err != nil {
		return false, err
	}

	createdGlobalnetRole, err := serviceaccount.EnsureRole(clientSet, namespace,
		embeddedyamls.ConfigRbacSubmarinerGlobalnetRole)
	if err != nil {
		return false, err
	}

	createdMetricsReaderRole, err := serviceaccount.EnsureRole(clientSet, namespace,
		embeddedyamls.ConfigRbacSubmarinerMetricsReaderRole)
	if err != nil {
		return false, err
	}

	return createdOperatorRole || createdSubmarinerRole || createdRouteAgentRole || createdGlobalnetRole || createdMetricsReaderRole, err
}

func ensureRoleBindings(clientSet *clientset.Clientset, namespace string) (bool, error) {
	createdOperatorRB, err := serviceaccount.EnsureRoleBinding(clientSet, namespace,
		embeddedyamls.ConfigRbacSubmarinerOperatorRoleBinding)
	if err != nil {
		return false, err
	}

	createdSubmarinerRB, err := serviceaccount.EnsureRoleBinding(clientSet, namespace,
		embeddedyamls.ConfigRbacSubmarinerGatewayRoleBinding)
	if err != nil {
		return false, err
	}

	createdRouteAgentRB, err := serviceaccount.EnsureRoleBinding(clientSet, namespace,
		embeddedyamls.ConfigRbacSubmarinerRouteAgentRoleBinding)
	if err != nil {
		return false, err
	}

	createdGlobalnetRB, err := serviceaccount.EnsureRoleBinding(clientSet, namespace,
		embeddedyamls.ConfigRbacSubmarinerGlobalnetRoleBinding)
	if err != nil {
		return false, err
	}

	createdMetricsReaderRB, err := serviceaccount.EnsureRoleBinding(clientSet, namespace,
		embeddedyamls.Config_rbac_submariner_metrics_reader_role_binding_yaml)
	if err != nil {
		return false, err
	}

	return createdOperatorRB || createdSubmarinerRB || createdRouteAgentRB || createdGlobalnetRB || createdMetricsReaderRB, err
}
