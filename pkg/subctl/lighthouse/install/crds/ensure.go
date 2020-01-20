/*
© 2020 Red Hat, Inc. and others.

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
	"fmt"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
)

//go:generate go run generators/yamls2go.go

// Copied over from operator/install/crds/ensure.go

//Ensure functions updates or installs the multiclusterservives CRDs in the cluster
func Ensure(restConfig *rest.Config) (bool, error) {
	clientSet, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	crd, err := getMcsCRD()
	if err != nil {
		return false, err
	}

	// Attempt to update or create the CRD definition
	// TODO(majopela): In the future we may want to report when we have updated the existing
	//                 CRD definition with new versions
	return updateOrCreateCRD(clientSet, crd)

}

func updateOrCreateCRD(clientSet clientset.Interface, crd *apiextensionsv1beta1.CustomResourceDefinition) (bool, error) {
	_, err := clientSet.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err == nil {
		return true, nil
	} else if errors.IsAlreadyExists(err) {
		existingCrd, err := clientSet.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.Name, v1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get pre-existing CRD %s : %s", crd.Name, err)
		}
		crd.ResourceVersion = existingCrd.ResourceVersion
		_, err = clientSet.ApiextensionsV1beta1().CustomResourceDefinitions().Update(crd)
		if err != nil {
			return false, fmt.Errorf("failed to update pre-existing CRD %s : %s", crd.Name, err)
		}
		return false, nil
	}
	return false, err
}

func getMcsCRD() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{}

	if err := embeddedyamls.GetObject(embeddedyamls.Lighthouse_crds_multiclusterservices_crd_yaml, crd); err != nil {
		return nil, err
	}

	return crd, nil
}
