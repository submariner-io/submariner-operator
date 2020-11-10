/*
© 2019 Red Hat, Inc. and others.

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

package engine

import (
	"fmt"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/utils"
	crdutils "github.com/submariner-io/submariner-operator/pkg/utils/crds"
)

// Ensure ensures that the required resources are deployed on the target system
// The resources handled here are the engine CRDs: Cluster and Endpoint
func Ensure(crdUpdater crdutils.CRDUpdater) error {
	clustersCrd, err := newClustersCRD()
	if err != nil {
		return fmt.Errorf("error creating the Cluster CRD: %s", err)
	}
	_, err = utils.CreateOrUpdateCRD(crdUpdater, clustersCrd)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("error provisioning the Cluster CRD: %s", err)
	}
	endpointsCrd, err := newEndpointsCRD()
	if err != nil {
		return fmt.Errorf("error creating the Endpoint CRD: %s", err)
	}
	_, err = utils.CreateOrUpdateCRD(crdUpdater, endpointsCrd)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("error provisioning the Endpoint CRD: %s", err)
	}
	gatewaysCrd, err := newGatewaysCRD()
	if err != nil {
		return fmt.Errorf("error creating the Gateway CRD: %s", err)
	}
	_, err = utils.CreateOrUpdateCRD(crdUpdater, gatewaysCrd)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("error provisioning the Gateway CRD: %s", err)
	}
	return nil
}

func newEndpointsCRD() (*apiextensions.CustomResourceDefinition, error) {
	crd := &apiextensions.CustomResourceDefinition{}

	if err := embeddedyamls.GetObject(embeddedyamls.Deploy_submariner_crds_submariner_io_endpoints_yaml, crd); err != nil {
		return nil, err
	}

	return crd, nil
}

func newClustersCRD() (*apiextensions.CustomResourceDefinition, error) {
	crd := &apiextensions.CustomResourceDefinition{}

	if err := embeddedyamls.GetObject(embeddedyamls.Deploy_submariner_crds_submariner_io_clusters_yaml, crd); err != nil {
		return nil, err
	}

	return crd, nil
}

func newGatewaysCRD() (*apiextensions.CustomResourceDefinition, error) {
	crd := &apiextensions.CustomResourceDefinition{}

	if err := embeddedyamls.GetObject(embeddedyamls.Deploy_submariner_crds_submariner_io_gateways_yaml, crd); err != nil {
		return nil, err
	}

	return crd, nil
}
