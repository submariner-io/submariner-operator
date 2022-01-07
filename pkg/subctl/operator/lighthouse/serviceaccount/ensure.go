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
	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/pkg/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/serviceaccount"
	"k8s.io/client-go/kubernetes"
)

// Ensure functions updates or installs the operator CRDs in the cluster.
func Ensure(kubeClient kubernetes.Interface, namespace string) (bool, error) {
	createdSA, err := ensureServiceAccounts(kubeClient, namespace)
	if err != nil {
		return false, err
	}

	createdCR, err := ensureClusterRoles(kubeClient)
	if err != nil {
		return false, err
	}

	createdCRB, err := ensureClusterRoleBindings(kubeClient, namespace)
	if err != nil {
		return false, err
	}

	return createdSA || createdCR || createdCRB, nil
}

func ensureServiceAccounts(kubeClient kubernetes.Interface, namespace string) (bool, error) {
	createdAgentSA, err := serviceaccount.Ensure(kubeClient, namespace,
		embeddedyamls.Config_rbac_lighthouse_agent_service_account_yaml)
	if err != nil {
		return false, errors.Wrap(err, "error provisioning the agent ServiceAccount resource")
	}

	createdCoreDNSSA, err := serviceaccount.Ensure(kubeClient, namespace,
		embeddedyamls.Config_rbac_lighthouse_coredns_service_account_yaml)

	return createdAgentSA || createdCoreDNSSA, errors.Wrap(err, "error provisioning the coredns ServiceAccount resource")
}

func ensureClusterRoles(kubeClient kubernetes.Interface) (bool, error) {
	createdAgentCR, err := serviceaccount.EnsureClusterRole(kubeClient,
		embeddedyamls.Config_rbac_lighthouse_agent_cluster_role_yaml)
	if err != nil {
		return false, errors.Wrap(err, "error provisioning the agent ClusterRole resource")
	}

	createdCoreDNSCR, err := serviceaccount.EnsureClusterRole(kubeClient,
		embeddedyamls.Config_rbac_lighthouse_coredns_cluster_role_yaml)

	return createdAgentCR || createdCoreDNSCR, errors.Wrap(err, "error provisioning the coredns ClusterRole resource")
}

func ensureClusterRoleBindings(kubeClient kubernetes.Interface, namespace string) (bool, error) {
	createdAgentCRB, err := serviceaccount.EnsureClusterRoleBinding(kubeClient, namespace,
		embeddedyamls.Config_rbac_lighthouse_agent_cluster_role_binding_yaml)
	if err != nil {
		return false, errors.Wrap(err, "error provisioning the agent ClusterRoleBinding resource")
	}

	createdCoreDNSCRB, err := serviceaccount.EnsureClusterRoleBinding(kubeClient, namespace,
		embeddedyamls.Config_rbac_lighthouse_coredns_cluster_role_binding_yaml)

	return createdAgentCRB || createdCoreDNSCRB, errors.Wrap(err, "error provisioning the coredns ClusterRoleBinding resource")
}
