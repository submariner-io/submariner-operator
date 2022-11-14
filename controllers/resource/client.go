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

package resource

import (
	"context"

	"github.com/submariner-io/admiral/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

//nolint:wrapcheck // These functions are pass-through wrappers for the k8s APIs.
func ForControllerClient(client controllerClient.Client, namespace string, objType controllerClient.Object) resource.Interface {
	return &resource.InterfaceFuncs{
		GetFunc: func(ctx context.Context, name string, options metav1.GetOptions) (runtime.Object, error) {
			obj := objType.DeepCopyObject()
			err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, obj.(controllerClient.Object))
			return obj, err
		},
		CreateFunc: func(ctx context.Context, obj runtime.Object, options metav1.CreateOptions) (runtime.Object, error) {
			err := client.Create(ctx, obj.(controllerClient.Object))
			return obj, err
		},
		UpdateFunc: func(ctx context.Context, obj runtime.Object, options metav1.UpdateOptions) (runtime.Object, error) {
			err := client.Update(ctx, obj.(controllerClient.Object))
			return obj, err
		},
		DeleteFunc: func(ctx context.Context, name string, options metav1.DeleteOptions) error {
			obj := objType.DeepCopyObject().(controllerClient.Object)
			obj.SetName(name)
			obj.SetNamespace(namespace)

			return client.Delete(ctx, obj)
		},
	}
}
