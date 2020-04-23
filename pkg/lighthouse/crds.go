package lighthouse

import (
	"fmt"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/utils"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/rest"
)

// Ensure ensures that the required resources are deployed on the target system
// The resources handled here are the lighthouse CRDs: MultiClusterService
func Ensure(config *rest.Config) error {
	clientSet, err := clientset.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating the api extensions client: %s", err)
	}
	mcsCrd, err := GetMcsCRD()
	if err != nil {
		return fmt.Errorf("error creating the MultiClusterService CRD: %s", err)
	}
	_, err = utils.CreateOrUpdateCRD(clientSet, mcsCrd)
	if err != nil {
		return fmt.Errorf("error creating the MultiClusterService CRD: %s", err)
	}
	return nil
}

func GetMcsCRD() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{}

	if err := embeddedyamls.GetObject(embeddedyamls.Lighthouse_crds_multiclusterservices_crd_yaml, crd); err != nil {
		return nil, err
	}

	return crd, nil
}
