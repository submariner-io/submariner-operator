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

	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
)

func All(clusterInfo *cluster.Info, status reporter.Interface) bool {
	success := Brokers(clusterInfo, status)

	fmt.Println()

	if clusterInfo.Submariner == nil {
		success = Versions(clusterInfo, status) && success

		fmt.Println()

		status.Warning(constants.SubmarinerNotInstalled)

		return success
	}

	success = Connections(clusterInfo, status) && success

	fmt.Println()

	success = Endpoints(clusterInfo, status) && success

	fmt.Println()

	success = Gateways(clusterInfo, status) && success

	fmt.Println()

	success = Network(clusterInfo, status) && success

	fmt.Println()

	success = Versions(clusterInfo, status) && success

	return success
}
