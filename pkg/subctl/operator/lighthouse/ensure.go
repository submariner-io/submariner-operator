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

package lighthouseop

import (
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/lighthouse/scc"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/lighthouse/serviceaccount"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func Ensure(status reporter.Interface, kubeClient kubernetes.Interface, dynClient dynamic.Interface,
	operatorNamespace string) (bool, error) {
	if created, err := serviceaccount.Ensure(kubeClient, operatorNamespace); err != nil {
		return created, err // nolint:wrapcheck // No need to wrap here
	} else if created {
		status.Success("Created lighthouse service account and role")
	}

	if created, err := scc.Ensure(dynClient, operatorNamespace); err != nil {
		return created, err // nolint:wrapcheck // No need to wrap here
	} else if created {
		status.Success("Updated the privileged SCC")
	}

	return true, nil
}
