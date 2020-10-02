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
	installedMCS, err := utils.CreateOrUpdateEmbeddedCRD(crdUpdater,
		embeddedyamls.Deploy_lighthouse_crds_lighthouse_submariner_io_multiclusterservices_yaml)
	if err != nil {
		return installedMCS, fmt.Errorf("Error creating the MultiClusterServices CRD: %s", err)
	}

	installedSI, err := utils.CreateOrUpdateEmbeddedCRD(crdUpdater,
		embeddedyamls.Deploy_lighthouse_crds_lighthouse_submariner_io_serviceimports_yaml)
	if err != nil {
		return installedSI, fmt.Errorf("Error creating the ServiceImport CRD: %s", err)
	}

	installedMCSSI, err := utils.CreateOrUpdateEmbeddedCRD(crdUpdater,
		embeddedyamls.Deploy_mcsapi_crds_multicluster_x_k8s_io_serviceimports_yaml)

	if err != nil {
		return installedMCSSI, fmt.Errorf("Error creating the MCS ServiceImport CRD: %s", err)
	}

	// The broker does not need the ServiceExport or ServiceDiscovery
	if isBroker {
		return installedMCS || installedMCSSI || installedSI, nil
	}

	installedSE, err := utils.CreateOrUpdateEmbeddedCRD(crdUpdater,
		embeddedyamls.Deploy_lighthouse_crds_lighthouse_submariner_io_serviceexports_yaml)

	if err != nil {
		return installedSE, fmt.Errorf("Error creating the ServiceExport CRD: %s", err)
	}

	installedMCSSE, err := utils.CreateOrUpdateEmbeddedCRD(crdUpdater,
		embeddedyamls.Deploy_mcsapi_crds_multicluster_x_k8s_io_serviceexports_yaml)

	if err != nil {
		return installedMCSSE, fmt.Errorf("Error creating the MCS ServiceExport CRD: %s", err)
	}

	installedSD, err := utils.CreateOrUpdateEmbeddedCRD(crdUpdater, embeddedyamls.Deploy_crds_submariner_io_servicediscoveries_yaml)
	if err != nil {
		return installedSD, err
	}

	return installedMCS || installedMCSSI || installedSI || installedSE || installedSD || installedMCSSE, nil
}
