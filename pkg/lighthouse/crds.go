package lighthouse

import (
	"fmt"

	"github.com/submariner-io/submariner-operator/pkg/utils"
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
	mcsCrd, err := utils.GetMcsCRD()
	if err != nil {
		return fmt.Errorf("error creating the MultiClusterService CRD: %s", err)
	}
	_, err = utils.CreateOrUpdateCRD(clientSet, mcsCrd)
	if err != nil {
		return fmt.Errorf("error creating the MultiClusterService CRD: %s", err)
	}
	return nil
}
