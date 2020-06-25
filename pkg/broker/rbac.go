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
	submarinerBrokerClusterRole      = "submariner-k8s-broker-cluster"
	submarinerBrokerAdminRole        = "submariner-k8s-broker-admin"
	SubmarinerBrokerAdminSA          = "submariner-k8s-broker-admin"
	submarinerBrokerClusterSAFmt     = "cluster-%s"
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

// Create a role to bind to Broker SA
func NewBrokerAdminRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: submarinerBrokerAdminRole,
		},
		Rules: []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				Verbs:     []string{"create", "get", "list", "watch", "patch", "update", "delete"},
				APIGroups: []string{"submariner.io"},
				Resources: []string{"clusters", "endpoints"},
			},
			rbacv1.PolicyRule{
				Verbs:     []string{"create", "get", "list", "patch", "update", "delete"},
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts", "secrets", "configmaps"},
			},
			rbacv1.PolicyRule{
				Verbs:     []string{"create", "get", "list", "delete"},
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"rolebindings"},
			},
			rbacv1.PolicyRule{
				Verbs:     []string{"create", "get", "list", "watch", "patch", "update", "delete"},
				APIGroups: []string{"lighthouse.submariner.io"},
				Resources: []string{"*"},
			},
		},
	}

}

// Create a role for each Cluster SAs to bind to
func NewBrokerClusterRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: submarinerBrokerClusterRole,
		},
		Rules: []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				Verbs:     []string{"create", "get", "list", "watch", "patch", "update", "delete"},
				APIGroups: []string{"submariner.io"},
				Resources: []string{"clusters", "endpoints"},
			},
			rbacv1.PolicyRule{
				Verbs:     []string{"create", "get", "list", "watch", "patch", "update", "delete"},
				APIGroups: []string{"lighthouse.submariner.io"},
				Resources: []string{"*"},
			},
		},
	}
}

// Create a role for to bind the cluster admin (subctl) SA
func NewBrokerRoleBinding(serviceAccount, role string) *rbacv1.RoleBinding {
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
			rbacv1.Subject{
				Namespace: "submariner-k8s-broker",
				Name:      serviceAccount,
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
