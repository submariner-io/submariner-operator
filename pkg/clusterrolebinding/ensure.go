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

package clusterrolebinding

import (
	"context"

	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/submariner-operator/pkg/embeddedyamls"
	resourceutil "github.com/submariner-io/submariner-operator/pkg/resource"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/kubernetes"
)

// nolint:wrapcheck // No need to wrap errors here.
func EnsureFromYAML(kubeClient kubernetes.Interface, namespace, yaml string) (bool, error) {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{}

	err := embeddedyamls.GetObject(yaml, clusterRoleBinding)
	if err != nil {
		return false, err
	}

	clusterRoleBinding.Subjects[0].Namespace = namespace

	return Ensure(kubeClient, clusterRoleBinding)
}

// nolint:wrapcheck // No need to wrap errors here.
func Ensure(kubeClient kubernetes.Interface, clusterRoleBinding *rbacv1.ClusterRoleBinding) (bool, error) {
	return resourceutil.CreateOrUpdate(context.TODO(), resource.ForClusterRoleBinding(kubeClient), clusterRoleBinding)
}
