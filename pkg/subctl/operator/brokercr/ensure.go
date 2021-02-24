/*
© 2021 Red Hat, Inc. and others.

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
	"github.com/submariner-io/admiral/pkg/resource"
	submarinerClientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/admiral/pkg/util"
	submariner "github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
)

const (
	BrokerName = "submariner-broker"
)

func Ensure(config *rest.Config, namespace string, brokerSpec submariner.BrokerSpec) error {
	brokerCR := &submariner.Broker{
		ObjectMeta: metav1.ObjectMeta{
			Name: BrokerName,
		},
		Spec: brokerSpec,
	}

	client, err := submarinerClientset.NewForConfig(config)
	if err != nil {
		return err
	}

	return util.CreateAnew(&resource.InterfaceFuncs{
		GetFunc: func(name string, options metav1.GetOptions) (runtime.Object, error) {
			return client.SubmarinerV1alpha1().Brokers(namespace).Get(name, options)
		},
		CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
			return client.SubmarinerV1alpha1().Brokers(namespace).Create(obj.(*submariner.Broker))
		},
		DeleteFunc: func(name string, options *metav1.DeleteOptions) error {
			return client.SubmarinerV1alpha1().Brokers(namespace).Delete(name, options)
		},
	}, brokerCR, nil)
}
