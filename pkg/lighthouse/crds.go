/*
© 2021 Red Hat, Inc. and others

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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
		return installedMCSSI || installedSI, nil
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

	return installedMCSSI || installedSI || installedSE || installedSD || installedMCSSE, nil
}
