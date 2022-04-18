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

package service

import (
	"context"

	"github.com/submariner-io/admiral/pkg/reporter"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	mcs "sigs.k8s.io/mcs-api/pkg/client/clientset/versioned/typed/apis/v1alpha1"
)

func Unexport(client *mcs.MulticlusterV1alpha1Client, namespace, svcName string, status reporter.Interface) error {
	err := client.ServiceExports(namespace).Delete(context.TODO(), svcName, metav1.DeleteOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return status.Error(err, "Service %s/%s was not previously exported", namespace, svcName)
		}

		return status.Error(err, "Failed to unexport Service")
	}

	status.Success("Service successfully unexported")

	return nil
}
