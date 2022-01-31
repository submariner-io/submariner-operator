/*
SPDX-License-Identifier: Apache-2.0

Copyright Contributors to the Submariner project.

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

package gateway

import (
	"context"

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/pkg/crd"
	"github.com/submariner-io/submariner-operator/pkg/embeddedyamls"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// Ensure ensures that the required resources are deployed on the target system.
// The resources handled here are the gateway CRDs: Cluster and Endpoint.
func Ensure(crdUpdater crd.Updater) error {
	_, err := crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(),
		embeddedyamls.Deploy_submariner_crds_submariner_io_clusters_yaml)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "error provisioning the Cluster CRD")
	}

	_, err = crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(),
		embeddedyamls.Deploy_submariner_crds_submariner_io_endpoints_yaml)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "error provisioning the Endpoint CRD")
	}

	_, err = crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(),
		embeddedyamls.Deploy_submariner_crds_submariner_io_gateways_yaml)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "error provisioning the Gateway CRD")
	}

	_, err = crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(),
		embeddedyamls.Deploy_submariner_crds_submariner_io_clusterglobalegressips_yaml)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "error provisioning the ClusterGlobalEgressIP CRD")
	}

	_, err = crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(),
		embeddedyamls.Deploy_submariner_crds_submariner_io_globalegressips_yaml)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "error provisioning the GlobalEgressIP CRD")
	}

	_, err = crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(),
		embeddedyamls.Deploy_submariner_crds_submariner_io_globalingressips_yaml)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "error provisioning the GlobalIngressIP CRD")
	}

	return nil
}
