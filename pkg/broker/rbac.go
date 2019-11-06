package broker

import (
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewBrokerNamespace() *v1.Namespace {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "submariner-k8s-broker",
		},
	}

	return ns
}

func NewBrokerSA() *v1.ServiceAccount {
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: "submariner-k8s-broker-client",
		},
	}

	return sa
}

func NewBrokerRole() *rbacv1.Role {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: "submariner-k8s-broker-client",
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
			Name: "submariner-k8s-broker-client",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "submariner-k8s-broker-client",
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Namespace: "submariner-k8s-broker",
				Name:      "submariner-k8s-broker-client",
				Kind:      "ServiceAccount",
			},
		},
	}

	return binding
}
