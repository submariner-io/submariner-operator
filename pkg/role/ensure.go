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

package role

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
	role := &rbacv1.Role{}

	err := embeddedyamls.GetObject(yaml, role)
	if err != nil {
		return false, err
	}

	return Ensure(kubeClient, namespace, role)
}

// nolint:wrapcheck // No need to wrap errors here.
func Ensure(kubeClient kubernetes.Interface, namespace string, role *rbacv1.Role) (bool, error) {
	return resourceutil.CreateOrUpdate(context.TODO(), resource.ForRole(kubeClient, namespace), role)
}
