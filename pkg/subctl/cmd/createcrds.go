package cmd

import (
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

func createEndpointsCRD() *apiextensions.CustomResourceDefinition {
	// Define endpoints.submariner.io CRD spec
	endpoints_crd_spec_names := apiextensions.CustomResourceDefinitionNames{Plural: "endpoints", Singular: "endpoint", ListKind: "EndpointList", Kind: "Endpoint"}
	endpoints_crd_spec_versions := apiextensions.CustomResourceDefinitionVersion{Name: "v1", Served: true, Storage: true}
	endpoints_crd_spec_conversion := apiextensions.CustomResourceConversion{Strategy: "None"}
	endpoints_crd_spec := apiextensions.CustomResourceDefinitionSpec{Group: "submariner.io", Names: endpoints_crd_spec_names, Scope: "Namespaced", Versions: []apiextensions.CustomResourceDefinitionVersion{endpoints_crd_spec_versions}, Version: "v1", Conversion: &endpoints_crd_spec_conversion}

	// Define endpoints.submariner.io CRD status
	endpoints_crd_status_names := apiextensions.CustomResourceDefinitionNames{Plural: "endpoints", Singular: "endpoint", ListKind: "EndpointList", Kind: "Endpoint"}
	endpoints_crd_status_storedversions := []string{"v1"}
	endpoints_crd_status:= apiextensions.CustomResourceDefinitionStatus{AcceptedNames: endpoints_crd_status_names, StoredVersions: endpoints_crd_status_storedversions}

	// Define endpoints.submariner.io CRD
	endpoints_crd := apiextensions.CustomResourceDefinition{Spec: endpoints_crd_spec, Status: endpoints_crd_status}

	return &endpoints_crd
}

func createClustersCRD() *apiextensions.CustomResourceDefinition {
	// Define clusters.submariner.io CRD spec
	clusters_crd_spec_names := apiextensions.CustomResourceDefinitionNames{Plural: "clusters", Singular: "cluster", ListKind: "ClusterList", Kind: "Cluster"}
	clusters_crd_spec_versions := apiextensions.CustomResourceDefinitionVersion{Name: "v1", Served: true, Storage: true}
	clusters_crd_spec_conversion := apiextensions.CustomResourceConversion{Strategy: "None"}
	clusters_crd_spec := apiextensions.CustomResourceDefinitionSpec{Group: "submariner.io", Names: clusters_crd_spec_names, Scope: "Namespaced", Versions: []apiextensions.CustomResourceDefinitionVersion{clusters_crd_spec_versions}, Version: "v1", Conversion: &clusters_crd_spec_conversion}

	// Define clusters.submariner.io CRD status
	clusters_crd_status_names := apiextensions.CustomResourceDefinitionNames{Plural: "clusters", Singular: "cluster", ListKind: "ClusterList", Kind: "Cluster"}
	clusters_crd_status_storedversions := []string{"v1"}
	clusters_crd_status:= apiextensions.CustomResourceDefinitionStatus{AcceptedNames: clusters_crd_status_names, StoredVersions: clusters_crd_status_storedversions}

	// Define clusters.submariner.io CRD
	clusters_crd := apiextensions.CustomResourceDefinition{Spec: clusters_crd_spec, Status: clusters_crd_status}

	return &clusters_crd
}
