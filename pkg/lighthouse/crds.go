package lighthouse

import (
	"fmt"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// Ensure ensures that the required resources are deployed on the target system
// The resources handled here are the lighthouse CRDs: MultiClusterService
func Ensure(config *rest.Config) error {
	apiext, err := clientset.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating the api extensions client: %s", err)
	}
	_, err = apiext.ApiextensionsV1beta1().CustomResourceDefinitions().Create(getMcsCRD())
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the MultiClusterService CRD: %s", err)
	}
	return nil
}

func getMcsCRD() *apiextensions.CustomResourceDefinition {
	crd := &apiextensions.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "multiclusterservices.lighthouse.submariner.io",
		},
		Spec: apiextensions.CustomResourceDefinitionSpec{
			Group: "lighthouse.submariner.io",
			Scope: apiextensions.NamespaceScoped,
			Names: apiextensions.CustomResourceDefinitionNames{
				Plural:   "multiclusterservices",
				Singular: "multiclusterservice",
				ListKind: "MultiClusterServiceList",
				Kind:     "MultiClusterService",
			},
			Version: "v1",
		},
	}

	return crd
}
