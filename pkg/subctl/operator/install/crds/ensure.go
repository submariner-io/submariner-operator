package crds

import (
	"fmt"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install/embeddedyamls"
)

//go:generate go run generators/yamls2go.go

//Ensure functions updates or installs the operator CRDs in the cluster
func Ensure(restConfig *rest.Config) (bool, error) {
	clientSet, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	crd, err := getSubmarinerCRD()
	if err != nil {
		return false, err
	}

	// Attempt to update or create the CRD definition
	// TODO(majopela): In the future we may want to report when we have updated the existing
	//                 CRD definition with new versions
	return updateOrCreateCRD(clientSet, crd)

}

func updateOrCreateCRD(clientSet *clientset.Clientset, crd *apiextensionsv1beta1.CustomResourceDefinition) (bool, error) {

	_, err := clientSet.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err == nil {
		return true, err
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
	}
	return false, nil
}

func getSubmarinerCRD() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{}

	if err := embeddedyamls.GetObject(embeddedyamls.Crds_submariner_io_submariners_crd_yaml, crd); err != nil {
		return nil, err
	}

	return crd, nil
}
