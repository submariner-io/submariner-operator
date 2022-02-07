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

package show

import (
	"fmt"

	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
)

func All(newCluster *cluster.Info, status reporter.Interface) bool {
	if newCluster.Submariner == nil {
		status.Warning(constants.SubmMissingMessage)

		return true
	}

	success := Connections(newCluster, status)

	fmt.Println()

	success = Endpoints(newCluster, status) && success

	fmt.Println()

	success = Gateways(newCluster, status) && success

	fmt.Println()

	success = Network(newCluster, status) && success

	fmt.Println()

	success = Versions(newCluster, status) && success

	return success
}
