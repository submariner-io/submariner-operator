package lighthouse

import (
	"fmt"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/utils"
	crdutils "github.com/submariner-io/submariner-operator/pkg/utils/crds"
)

const (
	BrokerCluster = true
	DataCluster   = false
)

// Ensure ensures that the required resources are deployed on the target system
// The resources handled here are the lighthouse CRDs: MultiClusterService,
// ServiceImport, ServiceExport and ServiceDiscovery
func Ensure(crdUpdater crdutils.CRDUpdater, isBroker bool) (bool, error) {
	installedSI, err := utils.CreateOrUpdateEmbeddedCRD(crdUpdater,
		embeddedyamls.Crds_lighthouse_submariner_io_serviceimports_crd_yaml)
	if err != nil {
		return installedSI, fmt.Errorf("Error creating the ServiceImport CRD: %s", err)
	}

	installedSE, err := utils.CreateOrUpdateEmbeddedCRD(crdUpdater,
		embeddedyamls.Crds_lighthouse_submariner_io_serviceexports_crd_yaml)

	if err != nil {
		return installedSE, fmt.Errorf("Error creating the ServiceExport CRD: %s", err)
	}

	installedSD, err := utils.CreateOrUpdateEmbeddedCRD(crdUpdater, embeddedyamls.Crds_submariner_io_servicediscoveries_crd_yaml)
	if err != nil {
		return installedSD, err
	}

	// The broker does not need the ServiceExport
	return isBroker || installedSE, nil
}
