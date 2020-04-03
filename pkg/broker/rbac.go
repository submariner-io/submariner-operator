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

package broker

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

const (
	submarinerBrokerRole = "submariner-k8s-broker-client"
)

func NewBrokerSA(submarinerBrokerSA string) *v1.ServiceAccount {
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: submarinerBrokerSA,
		},
	}

	return sa
}

// Create a role to bind to cluster specific SA
func NewSubctlBrokerRole(submarinerSubctlRole string) *rbacv1.Role {
	subctlrole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: submarinerSubctlRole,
		},
		Rules: []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				APIGroups: []string{"submariner.io"},
				Resources: []string{"clusters", "endpoints"},
				Verbs:     []string{"create", "get", "list", "watch", "patch", "update", "delete"},
			},
			rbacv1.PolicyRule{
				Verbs:     []string{"create", "get", "list", "delete"},
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts", "secrets"},
			},
			rbacv1.PolicyRule{
				Verbs:     []string{"create", "get", "list", "delete"},
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"roles", "rolebindings"},
			},
		},
	}

	return subctlrole
}

// Create a role for Broker SA to bind to
func NewClusterBrokerRole() *rbacv1.Role {
	clusterbrokerrole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: submarinerBrokerRole,
		},
		Rules: []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				APIGroups: []string{"submariner.io"},
				Resources: []string{"clusters", "endpoints"},
				Verbs:     []string{"create", "get", "list", "watch", "patch", "update", "delete"},
			},
			rbacv1.PolicyRule{
				Verbs:     []string{"create", "delete"},
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts"},
			},
			rbacv1.PolicyRule{
				Verbs:     []string{"create", "delete"},
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"roles", "rolebindings"},
			},
		},
	}

	return clusterbrokerrole
}

func NewBrokerRoleBinding(submarinerRole string, submarinerBrokerSA string) *rbacv1.RoleBinding {
	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: submarinerRole,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     submarinerRole,
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Namespace: "submariner-k8s-broker",
				Name:      submarinerBrokerSA,
				Kind:      "ServiceAccount",
			},
		},
	}

	return binding
}

func GetClientTokenSecret(clientSet clientset.Interface, brokerNamespace string, submarinerBrokerSA string) (*v1.Secret, error) {
	sa, err := clientSet.CoreV1().ServiceAccounts(brokerNamespace).Get(submarinerBrokerSA, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("ServiceAccount %s get failed: %s", submarinerBrokerSA, err)
	}
	if len(sa.Secrets) < 1 {
		return nil, fmt.Errorf("ServiceAccount %s does not have any secret", sa.Name)
	}
	brokerTokenPrefix := fmt.Sprintf("%s-token-", submarinerBrokerSA)

	for _, secret := range sa.Secrets {
		if strings.HasPrefix(secret.Name, brokerTokenPrefix) {
			return clientSet.CoreV1().Secrets(brokerNamespace).Get(secret.Name, metav1.GetOptions{})
		}
	}

	return nil, fmt.Errorf("ServiceAccount %s does not have a secret of type token", submarinerBrokerSA)
}
