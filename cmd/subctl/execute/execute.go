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

package execute

import (
	"fmt"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
	"os"

	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
)

func OnMultiCluster(restConfigProducer restconfig.Producer, run func(*cluster.Info, reporter.Interface) bool) {
	success := true

	status := cli.NewReporter()
	for _, config := range restConfigProducer.MustGetForClusters() {
		status.Start("Cluster %q", config.ClusterName)

		clientProducer, err := client.NewProducerFromRestConfig(config.Config)
		if err != nil {
			exit.OnErrorWithMessage(err, "Error creating the client producer")
		}

		newCluster, errMsg := cluster.New(config.ClusterName, clientProducer)
		if newCluster == nil {
			success = false

			status.Failure(errMsg.Error())
			fmt.Println()

			continue
		}

		success = run(newCluster, status) && success
		status.End()

		fmt.Println()
	}

	if !success {
		os.Exit(1)
	}
}
