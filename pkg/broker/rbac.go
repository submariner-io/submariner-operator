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

package broker

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	submarinerBrokerClusterRole      = "submariner-k8s-broker-cluster"
	submarinerBrokerAdminRole        = "submariner-k8s-broker-admin"
	submarinerBrokerClusterDefaultSA = "submariner-k8s-broker-client" // for backwards compatibility with documentation
)

func NewBrokerSA(submarinerBrokerSA string) *v1.ServiceAccount {
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: submarinerBrokerSA,
		},
	}

	return sa
}

// Create a role to bind to Broker SA.
func NewBrokerAdminRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: submarinerBrokerAdminRole,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"create", "get", "list", "watch", "patch", "update", "delete"},
				APIGroups: []string{"submariner.io"},
				Resources: []string{"clusters", "endpoints"},
			},
			{
				Verbs:     []string{"create", "get", "list", "update", "delete"},
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts", "secrets", "configmaps"},
			},
			{
				Verbs:     []string{"create", "get", "list", "delete"},
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"rolebindings"},
			},
			{
				Verbs:     []string{"create", "get", "list", "watch", "patch", "update", "delete"},
				APIGroups: []string{"multicluster.x-k8s.io"},
				Resources: []string{"*"},
			},
			{
				Verbs:     []string{"create", "get", "list", "watch", "patch", "update", "delete"},
				APIGroups: []string{"discovery.k8s.io"},
				Resources: []string{"endpointslices", "endpointslices/restricted"},
			},
		},
	}
}

// Create a role for each Cluster SAs to bind to.
func NewBrokerClusterRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: submarinerBrokerClusterRole,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"create", "get", "list", "watch", "patch", "update", "delete"},
				APIGroups: []string{"submariner.io"},
				Resources: []string{"clusters", "endpoints"},
			},
			{
				Verbs:     []string{"create", "get", "list", "watch", "patch", "update", "delete"},
				APIGroups: []string{"multicluster.x-k8s.io"},
				Resources: []string{"*"},
			},
			{
				Verbs:     []string{"create", "get", "list", "watch", "patch", "update", "delete"},
				APIGroups: []string{"discovery.k8s.io"},
				Resources: []string{"endpointslices", "endpointslices/restricted"},
			},
			{
				Verbs:     []string{"get", "list"},
				APIGroups: []string{""},
				Resources: []string{"secrets"},
			},
		},
	}
}

// Create a role for to bind the cluster admin (subctl) SA.
func NewBrokerRoleBinding(serviceAccount, role, namespace string) *rbacv1.RoleBinding {
	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", serviceAccount, role),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     role,
		},
		Subjects: []rbacv1.Subject{
			{
				Namespace: namespace,
				Name:      serviceAccount,
				Kind:      "ServiceAccount",
			},
		},
	}

	return binding
}
