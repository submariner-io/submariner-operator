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
	lighthouse "github.com/submariner-io/submariner-operator/pkg/lighthouse"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/utils"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/rest"
)

//go:generate go run generators/yamls2go.go

//Ensure functions updates or installs the operator CRDs in the cluster
func Ensure(restConfig *rest.Config) (bool, error) {
	clientSet, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	crd, err := getServiceDiscoveryCRD()
	if err != nil {
		return false, err
	}

	serviceDiscoveryResult, err := utils.CreateOrUpdateCRD(clientSet, crd)
	if err != nil {
		return serviceDiscoveryResult, err
	}

	mcsCrd, err := lighthouse.GetMcsCRD()
	if err != nil {
		return false, err
	}
	mcsCRDResult, err := utils.CreateOrUpdateCRD(clientSet, mcsCrd)
	if err != nil {
		return mcsCRDResult, err
	}

	return (serviceDiscoveryResult || mcsCRDResult), err
}

func getServiceDiscoveryCRD() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{}

	if err := embeddedyamls.GetObject(embeddedyamls.Crds_submariner_io_servicediscoveries_crd_yaml, crd); err != nil {
		return nil, err
	}

	return crd, nil
}
