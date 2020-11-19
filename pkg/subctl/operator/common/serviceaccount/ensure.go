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
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/utils"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

// Ensure creates the given service account
func Ensure(clientSet *clientset.Clientset, namespace, name string) (bool, error) {
	sa := &v1.ServiceAccount{ObjectMeta: v1meta.ObjectMeta{Name: name}}
	return utils.CreateOrUpdateServiceAccount(clientSet, namespace, sa)
}

func EnsureRole(clientSet *clientset.Clientset, namespace, yaml string) (bool, error) {
	role := &rbacv1.Role{}
	err := embeddedyamls.GetObject(yaml, role)
	if err != nil {
		return false, err
	}

	return utils.CreateOrUpdateRole(clientSet, namespace, role)
}

func EnsureRoleBinding(clientSet *clientset.Clientset, namespace, yaml string) (bool, error) {
	roleBinding := &rbacv1.RoleBinding{}
	err := embeddedyamls.GetObject(yaml, roleBinding)
	if err != nil {
		return false, err
	}

	return utils.CreateOrUpdateRoleBinding(clientSet, namespace, roleBinding)
}

func EnsureClusterRole(clientSet *clientset.Clientset, yaml string) (bool, error) {
	clusterRole := &rbacv1.ClusterRole{}
	err := embeddedyamls.GetObject(yaml, clusterRole)
	if err != nil {
		return false, err
	}

	return utils.CreateOrUpdateClusterRole(clientSet, clusterRole)
}

func EnsureClusterRoleBinding(clientSet *clientset.Clientset, namespace, yaml string) (bool, error) {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	err := embeddedyamls.GetObject(yaml, clusterRoleBinding)
	if err != nil {
		return false, err
	}

	clusterRoleBinding.Subjects[0].Namespace = namespace
	return utils.CreateOrUpdateClusterRoleBinding(clientSet, clusterRoleBinding)
}
