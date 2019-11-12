package broker

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

const (
	submarinerBrokerSA   = "submariner-k8s-broker-client"
	submarinerBrokerRole = "submariner-k8s-broker-client"
)

func NewBrokerSA() *v1.ServiceAccount {
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: submarinerBrokerSA,
		},
	}

	return sa
}

func NewBrokerRole() *rbacv1.Role {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: submarinerBrokerRole,
		},
		Rules: []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				APIGroups: []string{"submariner.io"},
				Resources: []string{"clusters", "endpoints"},
				Verbs:     []string{"create", "get", "list", "watch", "patch", "update", "delete"},
			},
		},
	}

	return role
}

func NewBrokerRoleBinding() *rbacv1.RoleBinding {
	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: submarinerBrokerRole,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     submarinerBrokerRole,
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

func GetClientTokenSecret(clientSet clientset.Interface, brokerNamespace string) (*v1.Secret, error) {
	sa, err := clientSet.CoreV1().ServiceAccounts(brokerNamespace).Get(submarinerBrokerSA, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("ServiceAccount %s get failed: %s", submarinerBrokerSA, err)
	}
	if len(sa.Secrets) < 1 {
		return nil, fmt.Errorf("ServiceAccount %s does not have any secret", sa.Name)
	}
	ref := sa.Secrets[0].Name
	return clientSet.CoreV1().Secrets(brokerNamespace).Get(ref, metav1.GetOptions{})
}
