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
	"os"

	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/internal/cli"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/cluster"
)

type OnClusterFn func(*cluster.Info, reporter.Interface) bool

func OnMultiCluster(restConfigProducer restconfig.Producer, run OnClusterFn) {
	restConfigs := restConfigProducer.MustGetForClusters()
	if len(restConfigs) == 0 {
		fmt.Println("No kube config was provided. Please use the --kubeconfig flag or set the KUBECONFIG environment variable")
		return
	}

	success := true
	status := cli.NewReporter()

	for _, config := range restConfigs {
		fmt.Printf("Cluster %q\n", config.ClusterName)

		clientProducer, err := client.NewProducerFromRestConfig(config.Config)
		if err != nil {
			status.Failure("Error creating the client producer: %v", err)
			fmt.Println()

			continue
		}

		clusterInfo, err := cluster.NewInfo(config.ClusterName, clientProducer, config.Config)
		if err != nil {
			success = false

			status.Failure("Error initializing the cluster information: %v", err)
			fmt.Println()

			continue
		}

		success = run(clusterInfo, status) && success

		fmt.Println()
	}

	if !success {
		os.Exit(1)
	}
}

func IfSubmarinerInstalled(run OnClusterFn) OnClusterFn {
	return func(clusterInfo *cluster.Info, status reporter.Interface) bool {
		if clusterInfo.Submariner == nil {
			status.Warning(constants.SubmarinerNotInstalled)

			return true
		}

		return run(clusterInfo, status)
	}
}
