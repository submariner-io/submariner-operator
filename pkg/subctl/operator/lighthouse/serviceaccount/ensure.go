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

package serviceaccount

import (
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/serviceaccount"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
)

const (
	LighthouseAgentServiceAccount   = "submariner-lighthouse-agent"
	LighthouseCoreDNSServiceAccount = "submariner-lighthouse-coredns"
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

	createdCR, err := ensureClusterRoles(clientSet)
	if err != nil {
		return false, err
	}

	createdCRB, err := ensureClusterRoleBindings(clientSet, namespace)
	if err != nil {
		return false, err
	}

	return createdSA || createdCR || createdCRB, nil
}

func ensureServiceAccounts(clientSet *clientset.Clientset, namespace string) (bool, error) {
	createdAgentSA, err := serviceaccount.Ensure(clientSet, namespace, LighthouseAgentServiceAccount)
	if err != nil {
		return false, err
	}

	createdCoreDNSSA, err := serviceaccount.Ensure(clientSet, namespace, LighthouseCoreDNSServiceAccount)
	if err != nil {
		return false, err
	}
	return createdAgentSA || createdCoreDNSSA, err
}

func ensureClusterRoles(clientSet *clientset.Clientset) (bool, error) {
	createdAgentCR, err := serviceaccount.EnsureClusterRole(clientSet, embeddedyamls.Config_rbac_lighthouse_agent_cluster_role_yaml)
	if err != nil {
		return false, err
	}

	createdCoreDNSCR, err := serviceaccount.EnsureClusterRole(clientSet, embeddedyamls.Config_rbac_lighthouse_coredns_cluster_role_yaml)
	if err != nil {
		return false, err
	}

	return createdAgentCR || createdCoreDNSCR, err
}

func ensureClusterRoleBindings(clientSet *clientset.Clientset, namespace string) (bool, error) {
	createdAgentCRB, err := serviceaccount.EnsureClusterRoleBinding(clientSet, namespace,
		embeddedyamls.Config_rbac_lighthouse_agent_cluster_role_binding_yaml)
	if err != nil {
		return false, err
	}

	createdCoreDNSCRB, err := serviceaccount.EnsureClusterRoleBinding(clientSet, namespace,
		embeddedyamls.Config_rbac_lighthouse_coredns_cluster_role_binding_yaml)
	if err != nil {
		return false, err
	}

	return createdAgentCRB || createdCoreDNSCRB, err
}
