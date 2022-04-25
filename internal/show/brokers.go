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

package show

import (
	"context"
	"fmt"
	"strings"

	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Brokers(clusterInfo *cluster.Info, status reporter.Interface) bool {
	template := "%-25.24s%-25.24s%-40.39s\n"

	status.Start("Detecting broker(s)")

	brokerList, err := clusterInfo.ClientProducer.ForOperator().SubmarinerV1alpha1().Brokers(corev1.NamespaceAll).List(
		context.TODO(), metav1.ListOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		status.Failure(err.Error())
		status.End()

		return false
	}

	status.End()

	brokers := brokerList.Items
	if len(brokers) == 0 {
		return true
	}

	fmt.Printf(template, "NAMESPACE", "NAME", "COMPONENTS")

	for i := range brokers {
		fmt.Printf(
			template,
			brokers[i].Namespace,
			brokers[i].Name,
			strings.Join(brokers[i].Spec.Components, ", "),
		)
	}

	return true
}
