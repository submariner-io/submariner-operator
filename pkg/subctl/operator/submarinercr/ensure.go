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

package submarinercr

import (
	"context"

	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/admiral/pkg/util"
	submariner "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	operatorclient "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	operatorv1alpha1client "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned/typed/submariner/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	SubmarinerName = "submariner"
)

func Ensure(client operatorclient.Interface, namespace string, submarinerSpec *submariner.SubmarinerSpec) error {
	submarinerCR := &submariner.Submariner{
		ObjectMeta: metav1.ObjectMeta{
			Name: SubmarinerName,
		},
		Spec: *submarinerSpec,
	}

	propagationPolicy := metav1.DeletePropagationForeground

	_, err := util.CreateAnew(context.TODO(), ResourceInterface(client.SubmarinerV1alpha1().Submariners(namespace)),
		submarinerCR, metav1.CreateOptions{}, metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})

	return errors.Wrap(err, "error creating Submariner resource")
}

// nolint:wrapcheck // No need to wrap.
func ResourceInterface(client operatorv1alpha1client.SubmarinerInterface) resource.Interface {
	return &resource.InterfaceFuncs{
		GetFunc: func(ctx context.Context, name string, options metav1.GetOptions) (runtime.Object, error) {
			return client.Get(ctx, name, options)
		},
		CreateFunc: func(ctx context.Context, obj runtime.Object, options metav1.CreateOptions) (runtime.Object, error) {
			return client.Create(ctx, obj.(*submariner.Submariner), options)
		},
		UpdateFunc: func(ctx context.Context, obj runtime.Object, options metav1.UpdateOptions) (runtime.Object, error) {
			return client.Update(ctx, obj.(*submariner.Submariner), options)
		},
		DeleteFunc: func(ctx context.Context, name string, options metav1.DeleteOptions) error {
			return client.Delete(ctx, name, options)
		},
	}
}
