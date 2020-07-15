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

	installedMCS, err := utils.CreateOrUpdateEmbeddedCRD(clientSet,
		embeddedyamls.Lighthouse_crds_multiclusterservices_crd_yaml)
	if err != nil {
		return installedMCS, fmt.Errorf("Error creating the MultiClusterServices CRD: %s", err)
	}

	installedSI, err := utils.CreateOrUpdateEmbeddedCRD(clientSet,
		embeddedyamls.Lighthouse_crds_serviceimport_crd_yaml)
	if err != nil {
		return installedSI, fmt.Errorf("Error creating the ServiceImport CRD: %s", err)
	}

	// The broker does not need the ServiceExport
	if isBroker {
		return installedMCS, nil
	}

	installedSE, err := utils.CreateOrUpdateEmbeddedCRD(clientSet,
		embeddedyamls.Lighthouse_crds_serviceexport_crd_yaml)

	if err != nil {
		return installedSE, fmt.Errorf("Error creating the ServiceExport CRD: %s", err)
	}

	return installedMCS || installedSE, nil
}
