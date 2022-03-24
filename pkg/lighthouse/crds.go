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

package lighthouse

import (
	"context"

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/pkg/crd"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	BrokerCluster = true
	DataCluster   = false
)

// Ensure ensures that the required resources are deployed on the target system
// The resources handled here are the lighthouse CRDs: MultiClusterService,
// ServiceImport, ServiceExport and ServiceDiscovery
// nolint:gocyclo // This really isn't complex and just trips the threshold.
func Ensure(crdUpdater crd.Updater, isBroker bool) (bool, error) {
	// Delete obsolete CRDs if they are still present
	err := crdUpdater.Delete(context.TODO(), "serviceimports.lighthouse.submariner.io", metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return false, errors.Wrap(err, "error deleting the obsolete ServiceImport CRD")
	}

	err = crdUpdater.Delete(context.TODO(), "serviceexports.lighthouse.submariner.io", metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return false, errors.Wrap(err, "error deleting the obsolete ServiceExport CRD")
	}

	err = crdUpdater.Delete(context.TODO(), "multiclusterservices.lighthouse.submariner.io", metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return false, errors.Wrap(err, "error deleting the obsolete MultiClusterServices CRD")
	}

	installedMCSSI, err := crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(), "mcsapi/crds/multicluster.x_k8s.io_serviceimports.yaml")
	if err != nil {
		return installedMCSSI, errors.Wrap(err, "error creating the MCS ServiceImport CRD")
	}

	// The broker does not need the ServiceExport or ServiceDiscovery
	if isBroker {
		return installedMCSSI, nil
	}

	installedMCSSE, err := crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(), "mcsapi/crds/multicluster.x_k8s.io_serviceexports.yaml")
	if err != nil {
		return installedMCSSI || installedMCSSE, errors.Wrap(err, "error creating the MCS ServiceExport CRD")
	}

	installedSD, err := crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(), "crds/submariner.io_servicediscoveries.yaml")
	if err != nil {
		return installedMCSSI || installedMCSSE || installedSD, errors.Wrap(err, "error creating the ServiceDiscovery CRD")
	}

	return installedMCSSI || installedMCSSE || installedSD, nil
}
