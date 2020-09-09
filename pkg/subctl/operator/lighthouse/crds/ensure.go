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
	"github.com/submariner-io/submariner-operator/pkg/lighthouse"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/utils"
	crdutils "github.com/submariner-io/submariner-operator/pkg/utils/crds"

	"fmt"

	"k8s.io/client-go/rest"
)

//go:generate go run generators/yamls2go.go

func Ensure(restConfig *rest.Config) (bool, error) {
	crdUpdater, err := crdutils.NewFromRestConfig(restConfig)
	if err != nil {
		return false, fmt.Errorf("error creating the api extensions client: %s", err)
	}

	serviceDiscoveryResult, err := utils.CreateOrUpdateEmbeddedCRD(crdUpdater, embeddedyamls.Crds_submariner_io_servicediscoveries_crd_yaml)
	if err != nil {
		return serviceDiscoveryResult, err
	}

	installed, err := lighthouse.Ensure(crdUpdater, lighthouse.DataCluster)
	if err != nil {
		return installed, err
	}

	return (serviceDiscoveryResult || installed), err
}
