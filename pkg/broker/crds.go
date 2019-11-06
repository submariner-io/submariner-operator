package broker

import (
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewEndpointsCRD() *apiextensions.CustomResourceDefinition {
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

func NewClustersCRD() *apiextensions.CustomResourceDefinition {
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
