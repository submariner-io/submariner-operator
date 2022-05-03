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

package cleanup

import (
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/cloud/aws"
)

var config aws.Config

func AWS(restConfigProducer *restconfig.Producer, status reporter.Interface) error {
	err := aws.RunOn(*restConfigProducer, &config, status,
		// nolint:wrapcheck // No need to wrap errors here
		func(cloud api.Cloud, gwDeployer api.GatewayDeployer, status reporter.Interface) error {
			err := gwDeployer.Cleanup(status)
			if err != nil {
				return err
			}

			return cloud.CleanupAfterSubmariner(status)
		})
	if err != nil {
		return status.Error(err, "Failed to cleanup AWS cloud")
	}

	return nil
}
