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

package kubefedop

import (
	"fmt"

	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/namespace"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/serviceaccount"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/kubefedop/clusterrole"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/kubefedop/clusterrolebinding"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/kubefedop/crds"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/kubefedop/deployment"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/kubefedop/role"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/kubefedop/rolebinding"
)

func Ensure(status *cli.Status, config *rest.Config, operatorNamespace string, operatorImage string, isController bool) error {

	if created, err := crds.Ensure(config); err != nil {
		return err
	} else if created {
		status.QueueSuccessMessage("Created operator CRDs")
	}

	if created, err := namespace.Ensure(config, operatorNamespace); err != nil {
		return err
	} else if created {
		status.QueueSuccessMessage(fmt.Sprintf("Created operator namespace: %s", operatorNamespace))
	}

	if created, err := serviceaccount.Ensure(config, operatorNamespace, "kubefed-operator"); err != nil {
		return err
	} else if created {
		status.QueueSuccessMessage("Created operator service account")
	}

	if created, err := clusterrole.Ensure(config); err != nil {
		return err
	} else if created {
		status.QueueSuccessMessage("Created operator cluster role")
	}

	if created, err := clusterrolebinding.Ensure(config); err != nil {
		return err
	} else if created {
		status.QueueSuccessMessage("Created operator cluster role binding")
	}

	if created, err := role.Ensure(config, operatorNamespace); err != nil {
		return err
	} else if created {
		status.QueueSuccessMessage("Created operator role")
	}

	if created, err := rolebinding.Ensure(config, operatorNamespace); err != nil {
		return err
	} else if created {
		status.QueueSuccessMessage("Created operator role binding")
	}

	if isController {
		if created, err := deployment.Ensure(config, operatorNamespace, operatorImage); err != nil {
			return err
		} else if created {
			status.QueueSuccessMessage("Deployed the operator successfully")
		}
	}

	return nil
}
