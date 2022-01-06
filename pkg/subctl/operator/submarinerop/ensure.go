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

package submarinerop

import (
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/crd"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/namespace"
	lighthouseop "github.com/submariner-io/submariner-operator/pkg/subctl/operator/lighthouse"
	opcrds "github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop/crds"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop/deployment"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop/scc"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop/serviceaccount"
)

// nolint:wrapcheck // No need to wrap errors here.
func Ensure(status reporter.Interface, clientProducer client.Producer, operatorNamespace, operatorImage string, debug bool) error {
	if created, err := opcrds.Ensure(crd.UpdaterFromClientSet(clientProducer.ForCRD())); err != nil {
		return err
	} else if created {
		status.Success("Created operator CRDs")
	}

	if created, err := namespace.Ensure(clientProducer.ForKubernetes(), operatorNamespace); err != nil {
		return err
	} else if created {
		status.Success("Created operator namespace: %s", operatorNamespace)
	}

	if created, err := serviceaccount.Ensure(clientProducer.ForKubernetes(), operatorNamespace); err != nil {
		return err
	} else if created {
		status.Success("Created operator service account and role")
	}

	if created, err := scc.Ensure(clientProducer.ForDynamic(), operatorNamespace); err != nil {
		return err
	} else if created {
		status.Success("Updated the privileged SCC")
	}

	if created, err := lighthouseop.Ensure(status, clientProducer.ForKubernetes(), clientProducer.ForDynamic(),
		operatorNamespace); err != nil {
		return err
	} else if created {
		status.Success("Created Lighthouse service accounts and roles")
	}

	if created, err := deployment.Ensure(clientProducer.ForKubernetes(), operatorNamespace, operatorImage, debug); err != nil {
		return err
	} else if created {
		status.Success("Deployed the operator successfully")
	}

	return nil
}
