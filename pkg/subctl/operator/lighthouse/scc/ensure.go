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
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/scc"
	embeddedyamls "github.com/submariner-io/submariner-operator/pkg/yamls"
	"k8s.io/client-go/rest"
)

func Ensure(restConfig *rest.Config, namespace string) (bool, error) {
	agentSaName, err := embeddedyamls.GetObjectName(embeddedyamls.Config_rbac_lighthouse_agent_service_account_yaml)
	if err != nil {
		return false, err
	}

	coreDNSSaName, err := embeddedyamls.GetObjectName(embeddedyamls.Config_rbac_lighthouse_coredns_service_account_yaml)
	if err != nil {
		return false, err
	}

	updateAgentSCC, err := scc.UpdateSCC(restConfig, namespace, agentSaName)
	if err != nil {
		return false, err
	}

	updateCoreDNSSCC, err := scc.UpdateSCC(restConfig, namespace, coreDNSSaName)
	if err != nil {
		return false, err
	}

	return updateAgentSCC || updateCoreDNSSCC, err
}
