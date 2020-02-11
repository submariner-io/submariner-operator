/*
© 2020 Red Hat, Inc. and others.

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

package install

import (
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/lighthouse/install/crds"
	"github.com/submariner-io/submariner-operator/pkg/subctl/lighthouse/install/deployment"
)

func Ensure(status *cli.Status, config *rest.Config, image string, isController bool) error {

	if created, err := crds.Ensure(config); err != nil {
		return err
	} else if created {
		status.QueueSuccessMessage("Created lighthouse CRDs")
	}

	if isController {
		//TODO: Install controller
		//TODO: kubefedctl enable MulticlusterService
		//TODO: kubectl --context=cluster1 patch clusterrole  -n kube-federation-system  kubefed-role  --type json -p "$(cat ${PRJ_ROOT}/scripts/kind-e2e/config/patch-kubefed-clusterrole.yaml)"
		//TODO: kubectl --context=cluster1 apply -f ${PRJ_ROOT}/package/lighthouse-controller-deployment.yaml
		if created, err := deployment.Ensure(config, "kubefed-operator", image); err != nil {
			return err
		} else if created {
			status.QueueSuccessMessage("Created lighthouse controller")
		}
	}

	return nil
}
