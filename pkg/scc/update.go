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

package scc

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/retry"
)

var openshiftSCCGVR = schema.GroupVersionResource{
	Group:    "security.openshift.io",
	Version:  "v1",
	Resource: "securitycontextconstraints",
}

func Update(dynClient dynamic.Interface, namespace, name string) (bool, error) {
	sccClient := dynClient.Resource(openshiftSCCGVR)

	created := false
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cr, err := sccClient.Get(context.TODO(), "privileged", metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return errors.Wrap(err, "error retrieving SCC resource")
		}
		users, found, err := unstructured.NestedSlice(cr.Object, "users")
		if !found || err != nil {
			return errors.Wrap(err, "error retrieving users field")
		}

		submarinerUser := fmt.Sprintf("system:serviceaccount:%s:%s", namespace, name)

		for _, user := range users {
			if submarinerUser == user.(string) {
				// the user is already part of the scc
				return nil
			}
		}

		if err := unstructured.SetNestedSlice(cr.Object, append(users, submarinerUser), "users"); err != nil {
			return errors.Wrap(err, "error setting users field")
		}

		if _, err = sccClient.Update(context.TODO(), cr, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "error updating OpenShift privileged SCC")
		}
		created = true
		return nil
	})

	return created, retryErr // nolint:wrapcheck // No need to wrap here
}
