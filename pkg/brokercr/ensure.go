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

package brokercr

import (
	"context"

	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/admiral/pkg/util"
	submariner "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	submarinerClientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	Name = "submariner-broker"
)

func Ensure(client submarinerClientset.Interface, namespace string, brokerSpec submariner.BrokerSpec) error {
	brokerCR := &submariner.Broker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name,
			Namespace: namespace,
		},
		Spec: brokerSpec,
	}

	// nolint:wrapcheck // No need to wrap errors here
	_, err := util.CreateAnew(context.TODO(), &resource.InterfaceFuncs{
		GetFunc: func(ctx context.Context, name string, options metav1.GetOptions) (runtime.Object, error) {
			return client.SubmarinerV1alpha1().Brokers(namespace).Get(ctx, name, options)
		},
		CreateFunc: func(ctx context.Context, obj runtime.Object, options metav1.CreateOptions) (runtime.Object, error) {
			return client.SubmarinerV1alpha1().Brokers(namespace).Create(ctx, obj.(*submariner.Broker), options)
		},
		DeleteFunc: func(ctx context.Context, name string, options metav1.DeleteOptions) error {
			return client.SubmarinerV1alpha1().Brokers(namespace).Delete(ctx, name, options)
		},
	}, brokerCR, metav1.CreateOptions{}, metav1.DeleteOptions{})

	return errors.Wrap(err, "error creating Broker resource")
}
