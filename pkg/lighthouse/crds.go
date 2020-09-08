package lighthouse

import (
	"fmt"

	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/utils"
)

const (
	BrokerCluster = true
	DataCluster   = false
)

// Ensure ensures that the required resources are deployed on the target system
// The resources handled here are the lighthouse CRDs: MultiClusterService and ServiceExport
func Ensure(config *rest.Config, isBroker bool) (bool, error) {
	clientSet, err := clientset.NewForConfig(config)
	if err != nil {
		return false, fmt.Errorf("error creating the api extensions client: %s", err)
	}

	installedSI, err := utils.CreateOrUpdateEmbeddedCRD(clientSet,
		embeddedyamls.Crds_lighthouse_submariner_io_serviceimports_crd_yaml)
	if err != nil {
		return installedSI, fmt.Errorf("Error creating the ServiceImport CRD: %s", err)
	}

	installedSE, err := utils.CreateOrUpdateEmbeddedCRD(clientSet,
		embeddedyamls.Crds_lighthouse_submariner_io_serviceexports_crd_yaml)

	if err != nil {
		return installedSE, fmt.Errorf("Error creating the ServiceExport CRD: %s", err)
	}

	// The broker does not need the ServiceExport
	return isBroker || installedSE, nil
}
