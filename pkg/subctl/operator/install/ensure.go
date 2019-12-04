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

package install

import (
	"fmt"

	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install/crds"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install/deployment"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install/namespace"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install/scc"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install/serviceaccount"
)

func Ensure(config *rest.Config, operatorNamespace string, operatorImage string) error {

	if created, err := crds.Ensure(config); err != nil {
		return err
	} else if created {
		fmt.Printf("* Created operator CRDs.\n")
	}

	if created, err := namespace.Ensure(config, operatorNamespace); err != nil {
		return err
	} else if created {
		fmt.Printf("* Created operator namespace: %s\n", operatorNamespace)
	}

	if created, err := serviceaccount.Ensure(config, operatorNamespace); err != nil {
		return err
	} else if created {
		fmt.Printf("* Created operator service account and role\n")
	}

	if created, err := scc.Ensure(config, operatorNamespace); err != nil {
		return err
	} else if created {
		fmt.Printf("* Updated the privileged SCC\n")
	}

	fmt.Printf("* Deploying the operator...\r")
	if created, err := deployment.Ensure(config, operatorNamespace, operatorImage); err != nil {
		return err
	} else if created {
		fmt.Printf("* Deployed the operator successfully\n")
	} else {
		fmt.Printf("                           \r")
	}

	fmt.Printf("* The operator is up and running\n")

	return nil
}
