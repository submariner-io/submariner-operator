/*
© 2020 Red Hat, Inc. and others.

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

package rolebinding

import (
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
)

//go:generate go run generators/yamls2go.go

//Ensure functions updates or installs the operator CRDs in the cluster
func Ensure(restConfig *rest.Config, namespace string) (bool, error) {
	clientSet, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

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
	err := embeddedyamls.GetObject(embeddedyamls.Kubefed_role_binding_yaml, roleBinding)
	if err != nil {
		return nil, err
	}
	return roleBinding, nil
}
