// NOTE: Boilerplate only.  Ignore this file.

// Package v2alpha1 contains API Schema definitions for the lighthouse v2alpha1 API group
// +k8s:deepcopy-gen=package,register
// +groupName=lighthouse.submariner.io
package v2alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "lighthouse.submariner.io", Version: "v2alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}
