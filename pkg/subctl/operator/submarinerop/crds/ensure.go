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

package crds

import (
	"context"

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/pkg/crd"
	"github.com/submariner-io/submariner-operator/pkg/embeddedyamls"
)

// Ensure functions updates or installs the operator CRDs in the cluster.
func Ensure(crdUpdater crd.Updater) (bool, error) {
	// Attempt to update or create the CRD definitions.
	// TODO(majopela): In the future we may want to report when we have updated the existing
	//                 CRD definition with new versions
	submarinerCreated, err := crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(),
		embeddedyamls.Deploy_crds_submariner_io_submariners_yaml)
	if err != nil {
		return false, errors.Wrap(err, "error provisioning Submariner CRD")
	}

	serviceDiscoveryCreated, err := crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(),
		embeddedyamls.Deploy_crds_submariner_io_servicediscoveries_yaml)
	if err != nil {
		return false, errors.Wrap(err, "error provisioning ServiceDiscovery CRD")
	}

	brokerCreated, err := crdUpdater.CreateOrUpdateFromEmbedded(context.TODO(),
		embeddedyamls.Deploy_crds_submariner_io_brokers_yaml)

	return submarinerCreated || serviceDiscoveryCreated || brokerCreated, errors.Wrap(err, "error provisioning Broker CRD")
}
