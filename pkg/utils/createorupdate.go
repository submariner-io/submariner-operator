/*
© 2021 Red Hat, Inc. and others.

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

package utils

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	crdutils "github.com/submariner-io/submariner-operator/pkg/utils/crds"
)

func CreateOrUpdateClusterRole(clientSet clientset.Interface, clusterRole *rbacv1.ClusterRole) (bool, error) {
	_, err := clientSet.RbacV1().ClusterRoles().Create(clusterRole)
	if err == nil {
		return true, nil
	} else if errors.IsAlreadyExists(err) {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			existingClusterRole, err := clientSet.RbacV1().ClusterRoles().Get(clusterRole.Name, v1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to retrieve pre-existing cluster role %s : %v", clusterRole.Name, err)
			}
			clusterRole.ResourceVersion = existingClusterRole.ResourceVersion
			// Potentially retried
			_, err = clientSet.RbacV1().ClusterRoles().Update(clusterRole)
			return err
		})
		return false, retryErr
	}
	return false, err
}

func CreateOrUpdateClusterRoleBinding(clientSet clientset.Interface, clusterRoleBinding *rbacv1.ClusterRoleBinding) (bool, error) {
	_, err := clientSet.RbacV1().ClusterRoleBindings().Create(clusterRoleBinding)
	if err == nil {
		return true, nil
	} else if errors.IsAlreadyExists(err) {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			existingClusterRoleBinding, err := clientSet.RbacV1().ClusterRoleBindings().Get(clusterRoleBinding.Name, v1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to retrieve pre-existing cluster role binding %s : %v", clusterRoleBinding.Name, err)
			}
			clusterRoleBinding.ResourceVersion = existingClusterRoleBinding.ResourceVersion
			// Potentially retried
			_, err = clientSet.RbacV1().ClusterRoleBindings().Update(clusterRoleBinding)
			return err
		})
		return false, retryErr
	}
	return false, err
}

func CreateOrUpdateCRD(updater crdutils.CRDUpdater, crd *apiextensions.CustomResourceDefinition) (bool, error) {
	_, err := updater.Create(crd)
	if err == nil {
		return true, nil
	} else if errors.IsAlreadyExists(err) {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			existingCrd, err := updater.Get(crd.Name, v1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to retrieve pre-existing CRD %s : %v", crd.Name, err)
			}
			crd.ResourceVersion = existingCrd.ResourceVersion
			// Potentially retried
			_, err = updater.Update(crd)
			return err
		})
		return false, retryErr
	}
	return false, err
}

func CreateOrUpdateEmbeddedCRD(updater crdutils.CRDUpdater, crdYaml string) (bool, error) {
	crd := &apiextensions.CustomResourceDefinition{}

	if err := embeddedyamls.GetObject(crdYaml, crd); err != nil {
		return false, fmt.Errorf("Error extracting embedded CRD: %s", err)
	}

	return CreateOrUpdateCRD(updater, crd)
}

func CreateOrUpdateDeployment(clientSet clientset.Interface, namespace string, deployment *appsv1.Deployment) (bool, error) {
	_, err := clientSet.AppsV1().Deployments(namespace).Create(deployment)
	if err != nil && errors.IsAlreadyExists(err) {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			existingDeployment, err := clientSet.AppsV1().Deployments(namespace).Get(deployment.Name, v1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to retrieve pre-existing deployment %s : %v", deployment.Name, err)
			}
			deployment.ResourceVersion = existingDeployment.ResourceVersion
			// Potentially retried
			_, err = clientSet.AppsV1().Deployments(namespace).Update(deployment)
			return err
		})
		return false, retryErr
	}
	return true, err
}

func CreateOrUpdateRole(clientSet clientset.Interface, namespace string, role *rbacv1.Role) (bool, error) {
	_, err := clientSet.RbacV1().Roles(namespace).Create(role)
	if err != nil && errors.IsAlreadyExists(err) {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			existingRole, err := clientSet.RbacV1().Roles(namespace).Get(role.Name, v1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to retrieve pre-existing role %s : %v", role.Name, err)
			}
			role.ResourceVersion = existingRole.ResourceVersion
			// Potentially retried
			_, err = clientSet.RbacV1().Roles(namespace).Update(role)
			return err
		})
		return false, retryErr
	}
	return true, err
}

func CreateOrUpdateRoleBinding(clientSet clientset.Interface, namespace string, roleBinding *rbacv1.RoleBinding) (bool, error) {
	_, err := clientSet.RbacV1().RoleBindings(namespace).Create(roleBinding)
	if err != nil && errors.IsAlreadyExists(err) {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			existingRoleBinding, err := clientSet.RbacV1().RoleBindings(namespace).Get(roleBinding.Name, v1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to retrieve pre-existing role binding %s : %v", roleBinding.Name, err)
			}
			roleBinding.ResourceVersion = existingRoleBinding.ResourceVersion
			// Potentially retried
			_, err = clientSet.RbacV1().RoleBindings(namespace).Update(roleBinding)
			return err
		})
		return false, retryErr
	}
	return true, err
}

func CreateOrUpdateServiceAccount(clientSet clientset.Interface, namespace string, sa *corev1.ServiceAccount) (bool, error) {
	_, err := clientSet.CoreV1().ServiceAccounts(namespace).Create(sa)
	if err != nil && errors.IsAlreadyExists(err) {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			existingSa, err := clientSet.CoreV1().ServiceAccounts(namespace).Get(sa.Name, v1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to retrieve pre-existing service account %s : %v", sa.Name, err)
			}
			sa.ResourceVersion = existingSa.ResourceVersion
			// Potentially retried
			_, err = clientSet.CoreV1().ServiceAccounts(namespace).Update(sa)
			return err
		})
		return false, retryErr
	}
	return true, err
}
