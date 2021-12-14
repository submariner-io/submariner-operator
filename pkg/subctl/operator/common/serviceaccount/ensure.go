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

package serviceaccount

import (
	"context"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/utils"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/kubernetes"
)

// Ensure creates the given service account.
// nolint:wrapcheck // No need to wrap errors here.
func Ensure(kubeClient kubernetes.Interface, namespace, yaml string) (bool, error) {
	sa := &v1.ServiceAccount{}

	err := embeddedyamls.GetObject(yaml, sa)
	if err != nil {
		return false, err
	}

	return utils.CreateOrUpdateServiceAccount(context.TODO(), kubeClient, namespace, sa)
}

// nolint:wrapcheck // No need to wrap errors here.
func EnsureRole(kubeClient kubernetes.Interface, namespace, yaml string) (bool, error) {
	role := &rbacv1.Role{}

	err := embeddedyamls.GetObject(yaml, role)
	if err != nil {
		return false, err
	}

	return utils.CreateOrUpdateRole(context.TODO(), kubeClient, namespace, role)
}

// nolint:wrapcheck // No need to wrap errors here.
func EnsureRoleBinding(kubeClient kubernetes.Interface, namespace, yaml string) (bool, error) {
	roleBinding := &rbacv1.RoleBinding{}

	err := embeddedyamls.GetObject(yaml, roleBinding)
	if err != nil {
		return false, err
	}

	return utils.CreateOrUpdateRoleBinding(context.TODO(), kubeClient, namespace, roleBinding)
}

// nolint:wrapcheck // No need to wrap errors here.
func EnsureClusterRole(kubeClient kubernetes.Interface, yaml string) (bool, error) {
	clusterRole := &rbacv1.ClusterRole{}

	err := embeddedyamls.GetObject(yaml, clusterRole)
	if err != nil {
		return false, err
	}

	return utils.CreateOrUpdateClusterRole(context.TODO(), kubeClient, clusterRole)
}

// nolint:wrapcheck // No need to wrap errors here.
func EnsureClusterRoleBinding(kubeClient kubernetes.Interface, namespace, yaml string) (bool, error) {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{}

	err := embeddedyamls.GetObject(yaml, clusterRoleBinding)
	if err != nil {
		return false, err
	}

	clusterRoleBinding.Subjects[0].Namespace = namespace

	return utils.CreateOrUpdateClusterRoleBinding(context.TODO(), kubeClient, clusterRoleBinding)
}
