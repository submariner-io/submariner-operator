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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// Ensure ensures that the required resources are deployed on the target system.
// The resources handled here are the gateway CRDs: Cluster and Endpoint.
func Ensure(crdUpdater crd.Updater) error {
	_, err := crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(), "submariner/crds/submariner.io_clusters.yaml")
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "error provisioning the Cluster CRD")
	}

	_, err = crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(), "submariner/crds/submariner.io_endpoints.yaml")
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "error provisioning the Endpoint CRD")
	}

	_, err = crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(), "submariner/crds/submariner.io_gateways.yaml")
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "error provisioning the Gateway CRD")
	}

	_, err = crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(), "submariner/crds/submariner.io_clusterglobalegressips.yaml")
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "error provisioning the ClusterGlobalEgressIP CRD")
	}

	_, err = crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(), "submariner/crds/submariner.io_globalegressips.yaml")
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "error provisioning the GlobalEgressIP CRD")
	}

	_, err = crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(), "submariner/crds/submariner.io_globalingressips.yaml")
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "error provisioning the GlobalIngressIP CRD")
	}

	return nil
}
