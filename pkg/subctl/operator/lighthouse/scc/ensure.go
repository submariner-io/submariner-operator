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

package scc

import (
	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/pkg/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/scc"
	"k8s.io/client-go/dynamic"
)

func Ensure(dynClient dynamic.Interface, namespace string) (bool, error) {
	agentSaName, err := embeddedyamls.GetObjectName(embeddedyamls.Config_rbac_lighthouse_agent_service_account_yaml)
	if err != nil {
		return false, errors.Wrap(err, "error parsing the agent ServiceAccount resource")
	}

	coreDNSSaName, err := embeddedyamls.GetObjectName(embeddedyamls.Config_rbac_lighthouse_coredns_service_account_yaml)
	if err != nil {
		return false, errors.Wrap(err, "error parsing the coredns ServiceAccount resource")
	}

	updateAgentSCC, err := scc.Update(dynClient, namespace, agentSaName)
	if err != nil {
		return false, errors.Wrap(err, "error updating the SCC resource")
	}

	updateCoreDNSSCC, err := scc.Update(dynClient, namespace, coreDNSSaName)

	return updateAgentSCC || updateCoreDNSSCC, errors.Wrap(err, "error updating the SCC resource")
}
