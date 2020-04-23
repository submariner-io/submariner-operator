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

package crds

import (
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/utils"
)

//go:generate go run generators/yamls2go.go

//Ensure functions updates or installs the operator CRDs in the cluster
func Ensure(restConfig *rest.Config) (bool, error) {
	clientSet, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	submarinerCrd, err := getSubmarinerCRD()
	if err != nil {
		return false, err
	}

	// Attempt to update or create the CRD definition
	// TODO(majopela): In the future we may want to report when we have updated the existing
	//                 CRD definition with new versions
	submarinerResult, err := utils.CreateOrUpdateCRD(clientSet, submarinerCrd)
	if err != nil {
		return submarinerResult, err
	}

	gatewaysResult, err := utils.CreateOrUpdateCRD(clientSet, getGatewaysCRD())
	return (submarinerResult || gatewaysResult), err
}

func getSubmarinerCRD() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{}

	if err := embeddedyamls.GetObject(embeddedyamls.Crds_submariner_io_submariners_crd_yaml, crd); err != nil {
		return nil, err
	}

	return crd, nil
}

// TODO Move this to the operator
func getGatewaysCRD() *apiextensionsv1beta1.CustomResourceDefinition {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: v1.ObjectMeta{
			Name: "gateways.submariner.io",
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group: "submariner.io",
			Scope: apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural:   "gateways",
				Singular: "gateway",
				ListKind: "GatewayList",
				Kind:     "Gateway",
			},
			Version: "v1",
		},
	}

	return crd
}
