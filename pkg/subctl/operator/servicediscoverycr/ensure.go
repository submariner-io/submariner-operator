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

package servicediscoverycr

import (
	"github.com/submariner-io/admiral/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	submariner "github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/names"
)

func init() {
	err := submariner.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
}

func Ensure(config *rest.Config, namespace string, serviceDiscoverySpec *submariner.ServiceDiscoverySpec) error {
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	client := dynClient.Resource(schema.GroupVersionResource{
		Group: "submariner.io", Version: "v1alpha1", Resource: "servicediscoveries"}).Namespace(namespace)

	sd := &submariner.ServiceDiscovery{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      names.ServiceDiscoveryCrName,
		},
		Spec: *serviceDiscoverySpec,
	}

	return createServiceDiscovery(client, sd)
}

func createServiceDiscovery(client dynamic.ResourceInterface, sd *submariner.ServiceDiscovery) error {
	serviceDiscovery, err := util.ToUnstructured(sd)
	if err != nil {
		return err
	}

	_, err = util.CreateOrUpdate(client, serviceDiscovery, func(existing *unstructured.Unstructured) (*unstructured.Unstructured, error) {
		err = unstructured.SetNestedField(existing.Object, util.GetNestedField(serviceDiscovery, "spec"), "spec")
		return existing, err
	})

	return err
}
