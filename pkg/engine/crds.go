package engine

import (
	"fmt"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// Ensure ensures that the required resources are deployed on the target system
// The resources handled here are the engine CRDs: Cluster and Endpoint
func Ensure(config *rest.Config) error {
	apiext, err := clientset.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating the api extensions client: %s", err)
	}
	_, err = apiext.ApiextensionsV1beta1().CustomResourceDefinitions().Create(newClustersCRD())
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the Cluster CRD: %s", err)
	}
	_, err = apiext.ApiextensionsV1beta1().CustomResourceDefinitions().Create(newEndpointsCRD())
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the Endpoint CRD: %s", err)
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
