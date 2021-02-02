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

	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/utils"
)

const (
	BrokerCluster = true
	DataCluster   = false
)

// Ensure ensures that the required resources are deployed on the target system
// The resources handled here are the lighthouse CRDs: MultiClusterService,
// ServiceImport, ServiceExport and ServiceDiscovery
func Ensure(cfg *rest.Config, isBroker bool) (bool, error) {

	// Delete obsolete CRDs if they are still present
	/* WIP: the patch won't be be ready without this, which is part of the upgrade path:
	we need some generic "utils.DeleteCRD" which will use the right deleter based on the cluster detected
	version
	err := crdUpdater.Delete("serviceimports.lighthouse.submariner.io", &metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return false, fmt.Errorf("Error deleting the obsolete ServiceImport CRD: %s", err)
	}
	err = crdUpdater.Delete("serviceexports.lighthouse.submariner.io", &metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return false, fmt.Errorf("Error deleting the obsolete ServiceExport CRD: %s", err)
	}
	err = crdUpdater.Delete("multiclusterservices.lighthouse.submariner.io", &metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return false, fmt.Errorf("Error deleting the obsolete MultiClusterServices CRD: %s", err)
	}
	*/
	installedMCSSI, err := utils.CreateOrUpdateEmbeddedCRD(cfg,
		embeddedyamls.Deploy_mcsapi_crds_multicluster_x_k8s_io_serviceimports_yaml)

	if err != nil {
		return installedMCSSI, fmt.Errorf("Error creating the MCS ServiceImport CRD: %s", err)
	}

	// The broker does not need the ServiceExport or ServiceDiscovery
	if isBroker {
		return installedMCSSI, nil
	}

	installedMCSSE, err := utils.CreateOrUpdateEmbeddedCRD(cfg,
		embeddedyamls.Deploy_mcsapi_crds_multicluster_x_k8s_io_serviceexports_yaml)

	if err != nil {
		return installedMCSSI || installedMCSSE, fmt.Errorf("Error creating the MCS ServiceExport CRD: %s", err)
	}

	installedSD, err := utils.CreateOrUpdateEmbeddedCRD(cfg, embeddedyamls.Deploy_crds_submariner_io_servicediscoveries_yaml)
	if err != nil {
		return installedMCSSI || installedMCSSE || installedSD, err
	}

	return installedMCSSI || installedMCSSE || installedSD, nil
}
