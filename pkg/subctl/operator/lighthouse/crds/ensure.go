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
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/utils"
	crdutils "github.com/submariner-io/submariner-operator/pkg/utils/crds"
)

func Ensure(restConfig *rest.Config) (bool, error) {
	crdUpdater, err := crdutils.NewFromRestConfig(restConfig)
	if err != nil {
		return false, errors.Wrap(err, "error creating the api extensions client")
	}

	return utils.CreateOrUpdateEmbeddedCRD(context.TODO(), crdUpdater, embeddedyamls.Deploy_crds_submariner_io_servicediscoveries_yaml)
}
