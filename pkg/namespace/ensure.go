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

package namespace

import (
	"context"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Ensure functions updates or installs the operator CRDs in the cluster.
func Ensure(kubeClient kubernetes.Interface, namespace string) (bool, error) {
	ns := &v1.Namespace{ObjectMeta: v1meta.ObjectMeta{Name: namespace}}

	_, err := kubeClient.CoreV1().Namespaces().Create(context.TODO(), ns, v1meta.CreateOptions{})

	if err == nil {
		return true, nil
	} else if apierrors.IsAlreadyExists(err) {
		return false, nil
	}

	return false, errors.Wrap(err, "error creating Namespace")
}
