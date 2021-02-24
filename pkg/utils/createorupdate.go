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

	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/admiral/pkg/util"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	crdutils "github.com/submariner-io/submariner-operator/pkg/utils/crds"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
)

func CreateOrUpdate(client resource.Interface, obj runtime.Object) (bool, error) {
	result, err := util.CreateOrUpdate(client, obj, util.Replace(obj))
	return result == util.OperationResultCreated, err
}

func CreateOrUpdateClusterRole(clientSet clientset.Interface, clusterRole *rbacv1.ClusterRole) (bool, error) {
	return CreateOrUpdate(resource.ForClusterRole(clientSet), clusterRole)
}

func CreateOrUpdateClusterRoleBinding(clientSet clientset.Interface, clusterRoleBinding *rbacv1.ClusterRoleBinding) (bool, error) {
	return CreateOrUpdate(resource.ForClusterRoleBinding(clientSet), clusterRoleBinding)
}

func CreateOrUpdateCRD(updater crdutils.CRDUpdater, crd *apiextensions.CustomResourceDefinition) (bool, error) {
	return CreateOrUpdate(&resource.InterfaceFuncs{
		GetFunc: func(name string, options metav1.GetOptions) (runtime.Object, error) {
			return updater.Get(name, options)
		},
		CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
			return updater.Create(obj.(*apiextensions.CustomResourceDefinition))
		},
		UpdateFunc: func(obj runtime.Object) (runtime.Object, error) {
			return updater.Update(obj.(*apiextensions.CustomResourceDefinition))
		},
	}, crd)
}

func CreateOrUpdateEmbeddedCRD(updater crdutils.CRDUpdater, crdYaml string) (bool, error) {
	crd := &apiextensions.CustomResourceDefinition{}

	if err := embeddedyamls.GetObject(crdYaml, crd); err != nil {
		return false, fmt.Errorf("error extracting embedded CRD: %s", err)
	}

	return CreateOrUpdateCRD(updater, crd)
}

func CreateOrUpdateDeployment(clientSet clientset.Interface, namespace string, deployment *appsv1.Deployment) (bool, error) {
	return CreateOrUpdate(resource.ForDeployment(clientSet, namespace), deployment)
}

func CreateOrUpdateRole(clientSet clientset.Interface, namespace string, role *rbacv1.Role) (bool, error) {
	return CreateOrUpdate(resource.ForRole(clientSet, namespace), role)
}

func CreateOrUpdateRoleBinding(clientSet clientset.Interface, namespace string, roleBinding *rbacv1.RoleBinding) (bool, error) {
	return CreateOrUpdate(resource.ForRoleBinding(clientSet, namespace), roleBinding)
}

func CreateOrUpdateServiceAccount(clientSet clientset.Interface, namespace string, sa *corev1.ServiceAccount) (bool, error) {
	return CreateOrUpdate(resource.ForServiceAccount(clientSet, namespace), sa)
}
