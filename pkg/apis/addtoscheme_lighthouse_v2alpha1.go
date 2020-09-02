package apis

import (
	"github.com/submariner-io/submariner-operator/pkg/apis/lighthouse/v2alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v2alpha1.SchemeBuilder.AddToScheme)
}
