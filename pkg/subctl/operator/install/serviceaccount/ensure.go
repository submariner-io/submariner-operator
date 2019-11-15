package serviceaccount

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install/embeddedyamls"
)

//go:generate go run generators/yamls2go.go

const OperatorServiceAccout = "submariner-operator"

//Ensure functions updates or installs the operator CRDs in the cluster
func Ensure(restConfig *rest.Config, namespace string) (bool, error) {
	clientSet, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	createdSa, err := ensureServiceAccount(clientSet, namespace)
	if err != nil {
		return false, err
	}

	upd, err := ensureRole(clientSet, namespace)
	if err != nil {
		return false, err
	}

	updRb, err := ensureRoleBinding(clientSet, namespace)
	if err != nil {
		return false, err
	}

	return createdSa || upd || updRb, err

}

func ensureServiceAccount(clientSet *clientset.Clientset, namespace string) (bool, error) {
	sa := &v1.ServiceAccount{ObjectMeta: v1meta.ObjectMeta{Name: OperatorServiceAccout}}
	_, err := clientSet.CoreV1().ServiceAccounts(namespace).Create(sa)
	if err == nil {
		return true, nil
	} else if errors.IsAlreadyExists(err) {
		return false, nil
	} else {
		return false, fmt.Errorf("ServiceAccount creation failed: %s", err)
	}

}

func ensureRole(clientSet *clientset.Clientset, namespace string) (bool, error) {
	role, err := getOperatorRole()
	if err != nil {
		return false, fmt.Errorf("Role update or create failed: %s", err)
	}

	return updateOrCreateRole(clientSet, namespace, role)

}

func updateOrCreateRole(clientSet *clientset.Clientset, namespace string, role *rbacv1.Role) (bool, error) {
	_, err := clientSet.RbacV1().Roles(namespace).Update(role)
	if err == nil {
		return false, nil
	} else if !errors.IsNotFound(err) {
		return false, err
	}
	_, err = clientSet.RbacV1().Roles(namespace).Create(role)
	return true, err
}

func ensureRoleBinding(clientSet *clientset.Clientset, namespace string) (bool, error) {
	roleBinding, err := getOperatorRoleBinding()
	if err != nil {
		return false, fmt.Errorf("RoleBinding update or create failed: %s", err)
	}
	return updateOrCreateRoleBinding(clientSet, namespace, roleBinding)
}

func updateOrCreateRoleBinding(clientSet *clientset.Clientset, namespace string, roleBinding *rbacv1.RoleBinding) (bool, error) {
	_, err := clientSet.RbacV1().RoleBindings(namespace).Update(roleBinding)
	if err == nil {
		return false, nil
	} else if !errors.IsNotFound(err) {
		return false, err
	}
	_, err = clientSet.RbacV1().RoleBindings(namespace).Create(roleBinding)
	return true, err
}

func getOperatorRoleBinding() (*rbacv1.RoleBinding, error) {

	roleBinding := &rbacv1.RoleBinding{}
	err := embeddedyamls.GetObject(embeddedyamls.Role_binding_yaml, roleBinding)
	if err != nil {
		return nil, err
	}
	return roleBinding, nil
}

func getOperatorRole() (*rbacv1.Role, error) {

	role := &rbacv1.Role{}
	err := embeddedyamls.GetObject(embeddedyamls.Role_yaml, role)
	if err != nil {
		return nil, err
	}
	return role, nil
}
