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
	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/submariner-operator/pkg/client"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	mcsv1a1 "sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

func Export(clientProducer client.Producer, serviceNamespace, svcName string, status reporter.Interface) error {
	_, err := clientProducer.ForKubernetes().CoreV1().Services(serviceNamespace).Get(context.TODO(), svcName, metav1.GetOptions{})
	if err != nil {
		return status.Error(err, "Unable to find the Service %q in namespace %q", svcName, serviceNamespace)
	}

	mcsServiceExport := &mcsv1a1.ServiceExport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: serviceNamespace,
		},
	}

	resourceServiceExport, err := resource.ToUnstructured(mcsServiceExport)
	if err != nil {
		return status.Error(err, "Failed to convert to Unstructured")
	}

	gvr := &schema.GroupVersionResource{
		Group:    mcsv1a1.GroupVersion.Group,
		Version:  mcsv1a1.GroupVersion.Version,
		Resource: "serviceexports",
	}

	_, err = clientProducer.ForDynamic().Resource(*gvr).Namespace(serviceNamespace).
		Create(context.TODO(), resourceServiceExport, metav1.CreateOptions{})
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			status.Success("Service already exported")
			return nil
		}

		return status.Error(err, "Failed to export Service")
	}

	status.Success("Service exported successfully")

	return nil
}
