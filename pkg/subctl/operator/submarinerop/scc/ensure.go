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

package scc

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop/serviceaccount"
)

var (
	openshiftSCCGVR = schema.GroupVersionResource{
		Group:    "security.openshift.io",
		Version:  "v1",
		Resource: "securitycontextconstraints",
	}
)

func Ensure(restConfig *rest.Config, namespace string) (bool, error) {

	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	sccClient := dynClient.Resource(openshiftSCCGVR)

	created := false
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cr, err := sccClient.Get("privileged", metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		users, found, err := unstructured.NestedSlice(cr.Object, "users")
		if !found || err != nil {
			return err
		}

		submarinerUser := fmt.Sprintf("system:serviceaccount:%s:%s", namespace, serviceaccount.OperatorServiceAccount)

		for _, user := range users {
			if submarinerUser == user.(string) {
				// the user is already part of the scc
				return nil
			}
		}

		if err = unstructured.SetNestedSlice(cr.Object, append(users, submarinerUser), "users"); err != nil {
			return err
		}

		if _, err = sccClient.Update(cr, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("Error updating OpenShift privileged SCC: %s", err)
		}
		created = true
		return nil
	})
	return created, retryErr
}
