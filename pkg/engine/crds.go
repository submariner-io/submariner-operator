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

package engine

import (
	"fmt"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/submariner-io/submariner-operator/pkg/utils"
	crdutils "github.com/submariner-io/submariner-operator/pkg/utils/crds"
)

// Ensure ensures that the required resources are deployed on the target system
// The resources handled here are the engine CRDs: Cluster and Endpoint
func Ensure(crdUpdater crdutils.CRDUpdater) error {
	_, err := utils.CreateOrUpdateCRD(crdUpdater, newClustersCRD())
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("error provisioning the Cluster CRD: %s", err)
	}
	_, err = utils.CreateOrUpdateCRD(crdUpdater, newEndpointsCRD())
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("error provisioning the Endpoint CRD: %s", err)
	}
	_, err = utils.CreateOrUpdateCRD(crdUpdater, newGatewaysCRD())
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("error provisioning the Gateway CRD: %s", err)
	}
	return nil
}

func newEndpointsCRD() *apiextensions.CustomResourceDefinition {
	crd := &apiextensions.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "endpoints.submariner.io",
		},
		Spec: apiextensions.CustomResourceDefinitionSpec{
			Group: "submariner.io",
			Scope: apiextensions.NamespaceScoped,
			Names: apiextensions.CustomResourceDefinitionNames{
				Plural:   "endpoints",
				Singular: "endpoint",
				ListKind: "EndpointList",
				Kind:     "Endpoint",
			},
			Version: "v1",
		},
	}

	return crd
}

func newClustersCRD() *apiextensions.CustomResourceDefinition {
	crd := &apiextensions.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "clusters.submariner.io",
		},
		Spec: apiextensions.CustomResourceDefinitionSpec{
			Group: "submariner.io",
			Scope: apiextensions.NamespaceScoped,
			Names: apiextensions.CustomResourceDefinitionNames{
				Plural:   "clusters",
				Singular: "cluster",
				ListKind: "ClusterList",
				Kind:     "Cluster",
			},
			Version: "v1",
		},
	}

	return crd
}

func newGatewaysCRD() *apiextensions.CustomResourceDefinition {
	crd := &apiextensions.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "gateways.submariner.io",
		},
		Spec: apiextensions.CustomResourceDefinitionSpec{
			Group: "submariner.io",
			Scope: apiextensions.NamespaceScoped,
			Names: apiextensions.CustomResourceDefinitionNames{
				Plural:   "gateways",
				Singular: "gateway",
				ListKind: "GatewayList",
				Kind:     "Gateway",
			},
			Version: "v1",
			AdditionalPrinterColumns: []apiextensions.CustomResourceColumnDefinition{
				{
					Name:        "ha-status",
					Type:        "string",
					Description: "High availability status of the Gateway",
					JSONPath:    ".status.haStatus",
				},
			},
		},
	}

	return crd
}
