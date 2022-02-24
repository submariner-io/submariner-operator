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

package servicediscoverycr

import (
	"context"

	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/resource"
	submariner "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	operatorClientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	operatorv1alpha1client "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned/typed/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/names"
	resourceutil "github.com/submariner-io/submariner-operator/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	err := submariner.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
}

func Ensure(client operatorClientset.Interface, namespace string, serviceDiscoverySpec *submariner.ServiceDiscoverySpec) error {
	sd := &submariner.ServiceDiscovery{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      names.ServiceDiscoveryCrName,
		},
		Spec: *serviceDiscoverySpec,
	}

	_, err := resourceutil.CreateOrUpdate(context.TODO(), ResourceInterface(client.SubmarinerV1alpha1().ServiceDiscoveries(namespace)), sd)

	return errors.Wrap(err, "error creating/updating ServiceDiscovery resource")
}

// nolint:wrapcheck // No need to wrap.
func ResourceInterface(client operatorv1alpha1client.ServiceDiscoveryInterface) resource.Interface {
	return &resource.InterfaceFuncs{
		GetFunc: func(ctx context.Context, name string, options metav1.GetOptions) (runtime.Object, error) {
			return client.Get(ctx, name, options)
		},
		CreateFunc: func(ctx context.Context, obj runtime.Object, options metav1.CreateOptions) (runtime.Object, error) {
			return client.Create(ctx, obj.(*submariner.ServiceDiscovery), options)
		},
		UpdateFunc: func(ctx context.Context, obj runtime.Object, options metav1.UpdateOptions) (runtime.Object, error) {
			return client.Update(ctx, obj.(*submariner.ServiceDiscovery), options)
		},
		DeleteFunc: func(ctx context.Context, name string, options metav1.DeleteOptions) error {
			return client.Delete(ctx, name, options)
		},
	}
}
