package utils

import (
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

func GetMcsCRD() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{}

	if err := embeddedyamls.GetObject(embeddedyamls.Lighthouse_crds_multiclusterservices_crd_yaml, crd); err != nil {
		return nil, err
	}

	return crd, nil
}
