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

package submarinerop

import (
	"fmt"

	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/namespace"
	lighthouseop "github.com/submariner-io/submariner-operator/pkg/subctl/operator/lighthouse"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop/crds"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop/deployment"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop/scc"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop/serviceaccount"
)

func Ensure(status *cli.Status, config *rest.Config, operatorNamespace string, operatorImage string) error {

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

	if created, err := serviceaccount.Ensure(config, operatorNamespace); err != nil {
		return err
	} else if created {
		status.QueueSuccessMessage("Created operator service account and role")
	}

	if created, err := scc.Ensure(config, operatorNamespace); err != nil {
		return err
	} else if created {
		status.QueueSuccessMessage("Updated the privileged SCC")
	}

	if created, err := lighthouseop.Ensure(status, config, operatorNamespace); err != nil {
		return err
	} else if created {
		status.QueueSuccessMessage("Created Lighthouse service accounts and roles")
	}

	if created, err := deployment.Ensure(config, operatorNamespace, operatorImage); err != nil {
		return err
	} else if created {
		status.QueueSuccessMessage("Deployed the operator successfully")
	}

	return nil
}
