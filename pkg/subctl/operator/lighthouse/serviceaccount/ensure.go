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
	"fmt"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/utils"
)

//go:generate go run generators/yamls2go.go

const (
	LighthouseServiceAccount = "submariner-lighthouse"
)

// Ensure functions updates or installs the operator CRDs in the cluster
func Ensure(restConfig *rest.Config, namespace string) (bool, error) {
	clientSet, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	createdSa, err := ensureServiceAccount(clientSet, namespace)
	if err != nil {
		return false, err
	}

	upCr, err := ensureClusterRole(clientSet, namespace)
	if err != nil {
		return false, err
	}

	upCrb, err := ensureClusterRoleBinding(clientSet, namespace)
	if err != nil {
		return false, err
	}

	return createdSa || upCr || upCrb, err
}

func ensureServiceAccount(clientSet *clientset.Clientset, namespace string) (bool, error) {
	sa := &v1.ServiceAccount{ObjectMeta: v1meta.ObjectMeta{Name: LighthouseServiceAccount}}
	return utils.CreateOrUpdateServiceAccount(clientSet, namespace, sa)
}

func ensureClusterRole(clientSet *clientset.Clientset, namespace string) (bool, error) {
	clusterRole, err := getOperatorClusterRole()
	if err != nil {
		return false, fmt.Errorf("ClusterRole update or create failed: %s", err)
	}

	return utils.CreateOrUpdateClusterRole(clientSet, clusterRole)
}

func ensureClusterRoleBinding(clientSet *clientset.Clientset, namespace string) (bool, error) {
	clusterRoleBinding, err := getOperatorClusterRoleBinding(namespace)
	if err != nil {
		return false, fmt.Errorf("clusterRoleBinding update or create failed: %s", err)
	}
	return utils.CreateOrUpdateClusterRoleBinding(clientSet, clusterRoleBinding)
}

func getOperatorClusterRoleBinding(namespace string) (*rbacv1.ClusterRoleBinding, error) {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	err := embeddedyamls.GetObject(embeddedyamls.Lighthouse_cluster_role_binding_yaml, clusterRoleBinding)
	if err != nil {
		return nil, err
	}
	clusterRoleBinding.Subjects[0].Namespace = namespace
	return clusterRoleBinding, nil
}

func getOperatorClusterRole() (*rbacv1.ClusterRole, error) {
	role := &rbacv1.ClusterRole{}
	err := embeddedyamls.GetObject(embeddedyamls.Lighthouse_cluster_role_yaml, role)
	if err != nil {
		return nil, err
	}
	return role, nil
}
