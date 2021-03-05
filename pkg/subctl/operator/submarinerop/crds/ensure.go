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

package crds

import (
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/utils"
	crdutils "github.com/submariner-io/submariner-operator/pkg/utils/crds"
)

// Ensure functions updates or installs the operator CRDs in the cluster
func Ensure(restConfig *rest.Config) (bool, error) {
	crdUpdater, err := crdutils.NewFromRestConfig(restConfig)
	if err != nil {
		return false, err
	}

	// Attempt to update or create the CRD definitions
	// TODO(majopela): In the future we may want to report when we have updated the existing
	//                 CRD definition with new versions
	submarinerCreated, err := utils.CreateOrUpdateEmbeddedCRD(crdUpdater, embeddedyamls.Deploy_crds_submariner_io_submariners_yaml)
	if err != nil {
		return false, err
	}
	brokerCreated, err := utils.CreateOrUpdateEmbeddedCRD(crdUpdater, embeddedyamls.Deploy_crds_submariner_io_brokers_yaml)
	return submarinerCreated || brokerCreated, err
}
