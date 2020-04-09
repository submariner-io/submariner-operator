// +build !ignore_autogenerated

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1alpha1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1.ServiceDiscovery":       schema_pkg_apis_submariner_v1alpha1_ServiceDiscovery(ref),
		"github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1.ServiceDiscoverySpec":   schema_pkg_apis_submariner_v1alpha1_ServiceDiscoverySpec(ref),
		"github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1.ServiceDiscoveryStatus": schema_pkg_apis_submariner_v1alpha1_ServiceDiscoveryStatus(ref),
		"github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1.Submariner":             schema_pkg_apis_submariner_v1alpha1_Submariner(ref),
		"github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1.SubmarinerSpec":         schema_pkg_apis_submariner_v1alpha1_SubmarinerSpec(ref),
		"github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1.SubmarinerStatus":       schema_pkg_apis_submariner_v1alpha1_SubmarinerStatus(ref),
	}
}

func schema_pkg_apis_submariner_v1alpha1_ServiceDiscovery(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ServiceDiscovery is the Schema for the servicediscoveries API",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1.ServiceDiscoverySpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1.ServiceDiscoveryStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1.ServiceDiscoverySpec", "github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1.ServiceDiscoveryStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_submariner_v1alpha1_ServiceDiscoverySpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ServiceDiscoverySpec defines the desired state of ServiceDiscovery",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"version": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"repository": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"brokerK8sCA": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"brokerK8sRemoteNamespace": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"brokerK8sApiServerToken": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"brokerK8sApiServer": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"debug": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"boolean"},
							Format: "",
						},
					},
					"clusterID": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"namespace": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
				},
				Required: []string{"brokerK8sCA", "brokerK8sRemoteNamespace", "brokerK8sApiServerToken", "brokerK8sApiServer", "debug", "clusterID", "namespace"},
			},
		},
	}
}

func schema_pkg_apis_submariner_v1alpha1_ServiceDiscoveryStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ServiceDiscoveryStatus defines the observed state of ServiceDiscovery",
				Type:        []string{"object"},
			},
		},
	}
}

func schema_pkg_apis_submariner_v1alpha1_Submariner(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "Submariner is the Schema for the submariners API",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1.SubmarinerSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1.SubmarinerStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1.SubmarinerSpec", "github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1.SubmarinerStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_submariner_v1alpha1_SubmarinerSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "SubmarinerSpec defines the desired state of Submariner",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"version": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"repository": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"ceIPSecNATTPort": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"ceIPSecIKEPort": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"ceIPSecDebug": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"boolean"},
							Format: "",
						},
					},
					"ceIPSecPSK": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"brokerK8sCA": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"brokerK8sRemoteNamespace": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"brokerK8sApiServerToken": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"brokerK8sApiServer": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"broker": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"natEnabled": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"boolean"},
							Format: "",
						},
					},
					"debug": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"boolean"},
							Format: "",
						},
					},
					"colorCodes": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"clusterID": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"serviceCIDR": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"clusterCIDR": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"globalCIDR": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"namespace": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"cableDriver": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"serviceDiscoveryEnabled": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"boolean"},
							Format: "",
						},
					},
				},
				Required: []string{"ceIPSecDebug", "ceIPSecPSK", "brokerK8sCA", "brokerK8sRemoteNamespace", "brokerK8sApiServerToken", "brokerK8sApiServer", "broker", "natEnabled", "debug", "clusterID", "serviceCIDR", "clusterCIDR", "namespace"},
			},
		},
	}
}

func schema_pkg_apis_submariner_v1alpha1_SubmarinerStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "SubmarinerStatus defines the observed state of Submariner",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"natEnabled": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"boolean"},
							Format: "",
						},
					},
					"colorCodes": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"clusterID": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"serviceCIDR": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"clusterCIDR": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"globalCIDR": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"cableDriver": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"engineDaemonSetStatus": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/api/apps/v1.DaemonSetStatus"),
						},
					},
					"routeAgentDaemonSetStatus": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/api/apps/v1.DaemonSetStatus"),
						},
					},
					"globalnetDaemonSetStatus": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/api/apps/v1.DaemonSetStatus"),
						},
					},
				},
				Required: []string{"natEnabled", "clusterID", "serviceCIDR", "clusterCIDR"},
			},
		},
		Dependencies: []string{
			"k8s.io/api/apps/v1.DaemonSetStatus"},
	}
}
